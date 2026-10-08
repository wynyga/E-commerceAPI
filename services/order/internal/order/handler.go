package order

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wynyga/ecommerce/shared/apperr"
	"github.com/wynyga/ecommerce/shared/auth"
	"github.com/wynyga/ecommerce/shared/httpx"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// Routes: every order route needs a signed-in user, and only ever sees that user's own orders.
func (h *Handler) Routes(r chi.Router, jwtKey []byte) {
	r.Route("/api/v1/orders", func(r chi.Router) {
		r.Use(auth.Authenticate(jwtKey))
		r.Get("/", h.list)
		r.Post("/", h.checkout)
		r.Get("/{id}", h.get)
		r.Post("/{id}/cancel", h.cancel)
	})
}

func userID(r *http.Request) (uint64, error) {
	c, ok := auth.FromContext(r.Context())
	if !ok {
		return 0, apperr.Unauthorized("not authenticated")
	}
	return uint64(c.UserID), nil
}

func orderID(r *http.Request) (uint64, error) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id == 0 {
		return 0, apperr.NotFound("order not found")
	}
	return id, nil
}

// checkout answers 201 for a new order and 200 when an Idempotency-Key replay returned the original.
func (h *Handler) checkout(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	o, created, err := h.svc.Checkout(r.Context(), uid, auth.BearerToken(r), r.Header.Get("Idempotency-Key"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	httpx.WriteJSON(w, status, o)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page := httpx.ParsePage(r)
	orders, total, err := h.svc.List(r.Context(), uid, page.Limit, page.Offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if orders == nil {
		orders = []Order{}
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		Items []Order `json:"items"`
		httpx.PageMeta
	}{orders, httpx.NewPageMeta(page, total)})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := orderID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	o, err := h.svc.Get(r.Context(), uid, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := orderID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	o, err := h.svc.Cancel(r.Context(), uid, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, o)
}
