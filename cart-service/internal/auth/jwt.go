package auth

import (
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// Claims sama seperti di auth-service
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// ValidateToken memvalidasi JWT dan mengembalikan klaimnya
func ValidateToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET not set in environment")
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		//Pastikan metode signin adalah HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
