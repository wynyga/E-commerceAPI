package cart

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

// Routes: every cart route needs a signed-in user.
func (h *Handler) Routes(r chi.Router, jwtKey []byte) {
	r.Route("/api/v1/cart", func(r chi.Router) {
		r.Use(auth.Authenticate(jwtKey))
		r.Get("/", h.get)
		r.Delete("/", h.clear)
		r.Post("/items", h.add)
		r.Patch("/items/{itemID}", h.setQuantity)
		r.Delete("/items/{itemID}", h.remove)
	})
}

func userID(r *http.Request) (uint64, error) {
	c, ok := auth.FromContext(r.Context())
	if !ok {
		return 0, apperr.Unauthorized("not authenticated")
	}
	return uint64(c.UserID), nil
}

func itemID(r *http.Request) (uint64, error) {
	id, err := strconv.ParseUint(chi.URLParam(r, "itemID"), 10, 64)
	if err != nil || id == 0 {
		return 0, apperr.NotFound("item not found")
	}
	return id, nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := h.svc.Get(r.Context(), uid)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) clear(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.Clear(r.Context(), uid); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	in, err := httpx.Decode[AddInput](r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := h.svc.AddItem(r.Context(), uid, in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) setQuantity(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := itemID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	in, err := httpx.Decode[QuantityInput](r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := h.svc.SetQuantity(r.Context(), uid, id, in.Quantity)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	uid, err := userID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := itemID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := h.svc.RemoveItem(r.Context(), uid, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}
