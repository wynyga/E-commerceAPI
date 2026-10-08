package product

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wynyga/ecommerce/shared/apperr"
	"github.com/wynyga/ecommerce/shared/auth"
	"github.com/wynyga/ecommerce/shared/httpx"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler { return &Handler{repo: repo} }

// Routes: reading is public, changing the catalogue needs the admin role, and the
// internal stock calls need the shared internal key (they are for the order service only).
func (h *Handler) Routes(r chi.Router, jwtKey []byte, internalKey string) {
	r.Route("/api/v1/products", func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
		r.Group(func(r chi.Router) {
			r.Use(auth.Authenticate(jwtKey), auth.RequireRole(auth.RoleAdmin))
			r.Post("/", h.create)
			r.Put("/{id}", h.update)
			r.Delete("/{id}", h.delete)
		})
	})
	r.Route("/internal/products", func(r chi.Router) {
		r.Use(httpx.InternalOnly(internalKey))
		r.Post("/{id}/reserve", h.reserve)
		r.Post("/{id}/release", h.release)
	})
}

func idParam(r *http.Request) (uint64, error) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id == 0 {
		return 0, apperr.NotFound("product not found")
	}
	return id, nil
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	page := httpx.ParsePage(r)
	items, total, err := h.repo.List(r.Context(), Query{
		Search: r.URL.Query().Get("q"), Sort: r.URL.Query().Get("sort"), Limit: page.Limit, Offset: page.Offset,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if items == nil {
		items = []Product{}
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		Items []Product `json:"items"`
		httpx.PageMeta
	}{items, httpx.NewPageMeta(page, total)})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := h.repo.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	in, err := httpx.Decode[Input](r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p := Product{Name: in.Name, Description: in.Description, Price: in.Price, Stock: in.Stock}
	if err := h.repo.Create(r.Context(), &p); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	in, err := httpx.Decode[Input](r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	p, err := h.repo.Update(r.Context(), id, in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reserve(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	in, err := httpx.Decode[QuantityInput](r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	res, err := h.repo.Reserve(r.Context(), id, in.Quantity)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handler) release(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	in, err := httpx.Decode[QuantityInput](r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.repo.Release(r.Context(), id, in.Quantity); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "released"})
}
