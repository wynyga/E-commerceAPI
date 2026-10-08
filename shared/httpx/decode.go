package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/wynyga/ecommerce/shared/apperr"
)

const maxBody = 1 << 20 // 1 MiB

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()
	// Report errors under the JSON field name the client actually sent.
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return f.Name
		}
		return name
	})
	return v
}

// Decode reads a JSON body into T, rejects unknown fields and oversized bodies, trims
// strings and validates `validate` tags. Failures are apperr.Invalid.
func Decode[T any](r *http.Request) (T, error) {
	var v T
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		var tooBig *http.MaxBytesError
		switch {
		case errors.As(err, &tooBig):
			return v, apperr.Invalid("request body too large")
		case errors.Is(err, io.EOF):
			return v, apperr.Invalid("request body is empty")
		default:
			return v, apperr.Invalid("invalid request body")
		}
	}
	trimStrings(reflect.ValueOf(&v).Elem())
	if err := Validate(v); err != nil {
		return v, err
	}
	return v, nil
}

// trimStrings trims whitespace on every string field (and []string element) of a struct,
// so "   " cannot satisfy `required`.
func trimStrings(v reflect.Value) {
	if v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		switch {
		case f.Kind() == reflect.String:
			f.SetString(strings.TrimSpace(f.String()))
		case f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.String:
			for j := 0; j < f.Len(); j++ {
				f.Index(j).SetString(strings.TrimSpace(f.Index(j).String()))
			}
		}
	}
}

// Validate checks struct tags and returns an apperr with per-field messages.
func Validate(v any) error {
	err := validate.Struct(v)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return apperr.Invalid("invalid request body")
	}
	fields := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		fields[fe.Field()] = describe(fe)
	}
	return apperr.InvalidFields(fields)
}

func describe(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "min":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("must be at least %s characters", fe.Param())
		}
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("must be at most %s characters", fe.Param())
		}
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "email":
		return "must be a valid email address"
	default:
		return "is invalid"
	}
}
