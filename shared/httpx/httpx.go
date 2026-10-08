// Package httpx holds the net/http plumbing every service shares: JSON responses, error
// mapping, request decoding with validation, pagination and the internal-key guard.
package httpx

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/wynyga/ecommerce/shared/apperr"
)

// Headers used between services.
const HeaderInternalKey = "X-Internal-Key"

type errorBody struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpx: write response: %v", err)
	}
}

// WriteError writes err as {"error": "..."} with the right status. Only apperr errors
// expose their message; anything else is logged and reported as a generic 500, so
// database details never reach clients.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		WriteJSON(w, apperr.Status(ae), errorBody{Error: ae.Message, Fields: ae.Fields})
		return
	}
	if status := apperr.Status(err); status != http.StatusInternalServerError {
		WriteJSON(w, status, errorBody{Error: err.Error()})
		return
	}
	log.Printf("ERROR %s %s: %v", r.Method, r.URL.Path, err)
	WriteJSON(w, http.StatusInternalServerError, errorBody{Error: "internal server error"})
}

// InternalOnly rejects requests that do not carry the shared internal key.
func InternalOnly(key string) func(http.Handler) http.Handler {
	want := []byte(key)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get(HeaderInternalKey)), want) != 1 {
				WriteError(w, r, apperr.Unauthorized("missing or invalid internal key"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Page is a parsed ?page=&limit= pair.
type Page struct {
	Page   int
	Limit  int
	Offset int
}

// ParsePage reads page (default 1) and limit (default 20, max 100); bad values are clamped.
func ParsePage(r *http.Request) Page {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return Page{Page: page, Limit: limit, Offset: (page - 1) * limit}
}

// PageMeta is the pagination block returned with lists.
type PageMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func NewPageMeta(p Page, total int) PageMeta {
	pages := 0
	if p.Limit > 0 {
		pages = (total + p.Limit - 1) / p.Limit
	}
	return PageMeta{Page: p.Page, Limit: p.Limit, Total: total, TotalPages: pages}
}

// Health answers 200 for liveness probes.
func Health(service string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": service})
	}
}
