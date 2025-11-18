package cart

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	// Impor paket auth kita!
	"cart-service/internal/auth"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes akan mendaftarkan semua route yang berhubungan dengan cart
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.handleGetCart)
	r.Post("/items", h.handleAddItem)
	r.Delete("/items/{itemID}", h.handleRemoveItem)
}

// Handler Method
func (h *Handler) handleGetCart(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil user ID dari context
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	// 2. Panggil service untuk mendapatkan cart
	cart, err := h.service.GetOrCreateCart(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// 3. Kembalikan response
	writeJSON(w, http.StatusOK, cart)
}

func (h *Handler) handleAddItem(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil userID
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Failed to get user from context")
		return
	}

	// 2. Decode JSON body
	var req AddItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// (Opsional: Tambahkan validasi untuk req.Quantity > 0 di sini)
	if req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "Quantity must be greater than zero")
		return
	}

	// 3. Panggil service
	cart, err := h.service.AddItemToCart(userID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 4. Kembalikan response
	writeJSON(w, http.StatusOK, cart)
}

func (h *Handler) handleRemoveItem(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil userID
	userID, ok := auth.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Failed to get user from context")
		return
	}

	// 2. Ambil {itemID} dari URL parameter
	itemIDStr := chi.URLParam(r, "itemID")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid item ID parameter")
		return
	}

	// 3. Panggil service
	cart, err := h.service.RemoveItemFromCart(userID, itemID)
	if err != nil {
		// Tangani error spesifik
		if err.Error() == "item not found" || err.Error() == "unauthorized: item does not belong to user" {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// 4. Kembalikan response
	writeJSON(w, http.StatusOK, cart)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
