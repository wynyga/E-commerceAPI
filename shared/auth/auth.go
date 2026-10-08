// Package auth signs and verifies the HS256 tokens issued by the auth service and checked by
// every other service, and provides the matching chi middleware.
package auth

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/wynyga/ecommerce/shared/apperr"
	"github.com/wynyga/ecommerce/shared/httpx"
)

const (
	RoleCustomer = "customer"
	RoleAdmin    = "admin"
)

type Claims struct {
	UserID uint
	Role   string
}

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// Sign returns a token for the user and its expiry.
func Sign(key []byte, userID uint, role string, ttl time.Duration) (string, time.Time, error) {
	exp := time.Now().Add(ttl)
	c := claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(uint64(userID), 10),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(key)
	return tok, exp, err
}

// Parse verifies the signature (HS256 only) and expiry.
func Parse(key []byte, token string) (Claims, error) {
	var c claims
	t, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return key, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Claims{}, err
	}
	id, convErr := strconv.ParseUint(c.Subject, 10, 64)
	if !t.Valid || convErr != nil || id == 0 {
		return Claims{}, errors.New("invalid token")
	}
	return Claims{UserID: uint(id), Role: c.Role}, nil
}

type ctxKey struct{}

// FromContext returns the signed-in user stored by Authenticate.
func FromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(Claims)
	return c, ok
}

// BearerToken returns the raw token from the Authorization header, or "".
func BearerToken(r *http.Request) string {
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return strings.TrimSpace(token)
}

// Authenticate rejects requests without a valid bearer token and stores the claims for handlers.
func Authenticate(key []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := BearerToken(r)
			if token == "" {
				httpx.WriteError(w, r, apperr.Unauthorized("missing bearer token"))
				return
			}
			c, err := Parse(key, token)
			if err != nil {
				httpx.WriteError(w, r, apperr.Unauthorized("invalid or expired token"))
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, c)))
		})
	}
}

// RequireRole must run after Authenticate.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := FromContext(r.Context())
			if !ok {
				httpx.WriteError(w, r, apperr.Unauthorized("not authenticated"))
				return
			}
			if c.Role != role {
				httpx.WriteError(w, r, apperr.Forbidden("this action needs the %s role", role))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
