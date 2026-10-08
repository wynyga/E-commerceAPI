// Package user holds the account model and its layers: repository, service, handler.
package user

import (
	"context"
	"time"
)

type User struct {
	ID           uint      `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never leaves the service
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// RegisterPayload is the body of POST /api/v1/register. bcrypt only reads the first 72
// bytes of a password, so longer ones are refused rather than silently truncated.
type RegisterPayload struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type LoginPayload struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,max=72"`
}

type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

// Repository methods return apperr.NotFound / apperr.Conflict for the matching failures.
type Repository interface {
	Create(ctx context.Context, u User) (uint, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, id uint) (User, error)
}

type Service interface {
	Register(ctx context.Context, p RegisterPayload) (User, error)
	Login(ctx context.Context, p LoginPayload) (LoginResult, error)
	Profile(ctx context.Context, id uint) (User, error)
	// EnsureAdmin creates the admin account if the email is not registered yet.
	EnsureAdmin(ctx context.Context, email, password string) error
}
