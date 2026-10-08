// Package apperr defines the error categories shared by every service.
//
// Services return these from their service layer; the HTTP layer maps them to status
// codes in one place (httpx.WriteError), so handlers never pick a status by hand and
// internal errors never leak to clients.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Category sentinels. Use errors.Is(err, apperr.ErrNotFound) to test for them.
var (
	ErrInvalid      = errors.New("invalid request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrUnavailable  = errors.New("service unavailable")
)

// Error is a client-safe error: Message is shown to the caller as-is.
type Error struct {
	Kind    error
	Message string
	// Fields carries per-field validation messages for ErrInvalid.
	Fields map[string]string
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Kind }

func newErr(kind error, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

func Invalid(format string, args ...any) error      { return newErr(ErrInvalid, format, args...) }
func Unauthorized(format string, args ...any) error { return newErr(ErrUnauthorized, format, args...) }
func Forbidden(format string, args ...any) error    { return newErr(ErrForbidden, format, args...) }
func NotFound(format string, args ...any) error     { return newErr(ErrNotFound, format, args...) }
func Conflict(format string, args ...any) error     { return newErr(ErrConflict, format, args...) }
func Unavailable(format string, args ...any) error  { return newErr(ErrUnavailable, format, args...) }

// InvalidFields builds a validation error with per-field messages.
func InvalidFields(fields map[string]string) error {
	e := newErr(ErrInvalid, "validation failed")
	e.Fields = fields
	return e
}

// Status maps an error to its HTTP status. Unknown errors are 500.
func Status(err error) int {
	switch {
	case errors.Is(err, ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrUnavailable):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}
