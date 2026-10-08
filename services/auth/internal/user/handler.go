package user

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"github.com/wynyga/ecommerce/shared/apperr"
	"github.com/wynyga/ecommerce/shared/auth"
	"github.com/wynyga/ecommerce/shared/httpx"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// Routes mounts /register and /login (rate limited per client address, against password
// guessing) and /profile (needs a token).
func (h *Handler) Routes(r chi.Router, jwtKey []byte, perMinute int) {
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(perMinute, time.Minute))
		r.Post("/register", h.register)
		r.Post("/login", h.login)
	})
	r.With(auth.Authenticate(jwtKey)).Get("/profile", h.profile)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	p, err := httpx.Decode[RegisterPayload](r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	u, err := h.svc.Register(r.Context(), p)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"user": u})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	p, err := httpx.Decode[LoginPayload](r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	res, err := h.svc.Login(r.Context(), p)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) profile(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		httpx.WriteError(w, r, apperr.Unauthorized("not authenticated"))
		return
	}
	u, err := h.svc.Profile(r.Context(), claims.UserID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"user": u})
}
