// Komponen ini menghubungkan request HTTP dari luar dengan logika bisnis di dalam aplikasi.// internal/user/handler.go
package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler mengelola endpoint-endpoint terkait user
type Handler struct {
	service Service
}

// NewHandler membuat instance handler baru
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterUser menangani request registrasi pengguna
func (h *Handler) RegisterUser(c *gin.Context) {
	var payload RegisterPayload

	// Binding JSON dari request ke struct payload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Panggil service untuk mendaftarkan user
	user, err := h.service.RegisterUser(payload)
	if err != nil {
		// Di dunia nyata, Anda harus menangani error duplikat email secara spesifik
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Kirim response sukses
	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user":    user,
	})
}

// LoginUser menangani request login pengguna
func (h *Handler) LoginUser(c *gin.Context) {
	var payload LoginPayLoad

	// Binding JSON dari request ke struct payload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Panggil service untuk login user
	token, err := h.service.LoginUser(payload)
	if err != nil {
		// Biasanya error "invalid credentials" akan mengembalikan status 401 Unauthorized
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Kirim response sukses dengan token JWT
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
	})
}
