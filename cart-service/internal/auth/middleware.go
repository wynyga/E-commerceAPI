package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware memeriksa header Authorization untuk token JWT yang valid
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Mengambil token dari header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}
		//Mengambil token dari header "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		//Memvalidasi token
		claims, err := ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}
		//Menambahkan userID ke context request
		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		//Lanjutkan ke handler berikutnya dengan context yang diperbarui
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
