package shared_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/wynyga/ecommerce/shared/apperr"
	"github.com/wynyga/ecommerce/shared/auth"
	"github.com/wynyga/ecommerce/shared/httpx"
)

var key = []byte("test-key-that-is-long-enough-for-hs256-use")

// ---- apperr / httpx ---------------------------------------------------------

func TestStatusMapping(t *testing.T) {
	cases := map[error]int{
		apperr.Invalid("x"):      http.StatusBadRequest,
		apperr.Unauthorized("x"): http.StatusUnauthorized,
		apperr.Forbidden("x"):    http.StatusForbidden,
		apperr.NotFound("x"):     http.StatusNotFound,
		apperr.Conflict("x"):     http.StatusConflict,
		apperr.Unavailable("x"):  http.StatusBadGateway,
		errors.New("anything"):   http.StatusInternalServerError,
	}
	for err, want := range cases {
		if got := apperr.Status(err); got != want {
			t.Errorf("Status(%v) = %d, want %d", err, got, want)
		}
	}
}

func TestInternalErrorsNeverReachTheClient(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.WriteError(rec, httptest.NewRequest("GET", "/", nil), errors.New(`Error 1045: Access denied for user 'root'@'10.0.0.5'`))

	if rec.Code != 500 || strings.Contains(rec.Body.String(), "1045") || strings.Contains(rec.Body.String(), "root") {
		t.Errorf("response leaked internals: %d %s", rec.Code, rec.Body)
	}
}

type payload struct {
	Name  string `json:"name" validate:"required,max=10"`
	Price int64  `json:"price" validate:"gt=0"`
	Stock int    `json:"stock" validate:"gte=0"`
}

func decode(body string) (payload, error) {
	return httpx.Decode[payload](httptest.NewRequest("POST", "/", strings.NewReader(body)))
}

func TestDecodeTrimsAndValidates(t *testing.T) {
	p, err := decode(`{"name":"  Mug  ","price":5000,"stock":0}`)
	if err != nil || p.Name != "Mug" {
		t.Fatalf("decode = %+v, %v", p, err)
	}

	_, err = decode(`{"name":"   ","price":0,"stock":-1}`)
	var ae *apperr.Error
	if !errors.As(err, &ae) || len(ae.Fields) != 3 {
		t.Fatalf("err = %v, want 3 field errors (name blank, price, stock)", err)
	}
	for _, field := range []string{"name", "price", "stock"} { // keyed by the JSON names the client sent
		if _, ok := ae.Fields[field]; !ok {
			t.Errorf("missing field error for %q: %v", field, ae.Fields)
		}
	}
}

func TestDecodeRejectsUnknownFieldsAndBadBodies(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field": `{"name":"a","price":1,"stock":1,"admin":true}`,
		"not json":      `nope`,
		"empty":         ``,
		"wrong type":    `{"name":"a","price":"free","stock":1}`,
	} {
		if _, err := decode(body); !errors.Is(err, apperr.ErrInvalid) {
			t.Errorf("%s: err = %v, want invalid", name, err)
		}
	}
}

func TestInternalOnlyGuard(t *testing.T) {
	h := httpx.InternalOnly("secret-internal-key")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	for header, want := range map[string]int{"": 401, "wrong": 401, "secret-internal-key": 200} {
		req := httptest.NewRequest("POST", "/", nil)
		if header != "" {
			req.Header.Set(httpx.HeaderInternalKey, header)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Errorf("key %q -> %d, want %d", header, rec.Code, want)
		}
	}
}

func TestParsePageClampsValues(t *testing.T) {
	for query, want := range map[string]httpx.Page{
		"":                   {Page: 1, Limit: 20, Offset: 0},
		"?page=3&limit=10":   {Page: 3, Limit: 10, Offset: 20},
		"?page=-4&limit=0":   {Page: 1, Limit: 20, Offset: 0},
		"?limit=100000":      {Page: 1, Limit: 100, Offset: 0},
		"?page=abc&limit=xx": {Page: 1, Limit: 20, Offset: 0},
	} {
		if got := httpx.ParsePage(httptest.NewRequest("GET", "/"+query, nil)); got != want {
			t.Errorf("%q -> %+v, want %+v", query, got, want)
		}
	}
}

// ---- auth -------------------------------------------------------------------

func TestTokenRoundTrip(t *testing.T) {
	tok, _, err := auth.Sign(key, 42, auth.RoleAdmin, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	c, err := auth.Parse(key, tok)
	if err != nil || c.UserID != 42 || c.Role != auth.RoleAdmin {
		t.Fatalf("Parse = %+v, %v", c, err)
	}
}

func TestBadTokensAreRejected(t *testing.T) {
	good, _, _ := auth.Sign(key, 1, auth.RoleCustomer, time.Hour)
	expired, _, _ := auth.Sign(key, 1, auth.RoleCustomer, -time.Minute)
	wrongKey, _, _ := auth.Sign([]byte("a-completely-different-key-of-sufficient-length"), 1, auth.RoleCustomer, time.Hour)
	// A token signed with "none" must never be accepted, whatever it claims.
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "1", "role": "admin", "exp": time.Now().Add(time.Hour).Unix()}).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	noSub, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"role": "admin", "exp": time.Now().Add(time.Hour).Unix()}).SignedString(key)

	for name, tok := range map[string]string{
		"expired": expired, "wrong key": wrongKey, "alg none": none, "no subject": noSub,
		"tampered": good[:len(good)-3] + "xyz", "garbage": "not.a.jwt", "empty": "",
	} {
		if _, err := auth.Parse(key, tok); err == nil {
			t.Errorf("%s: token was accepted", name)
		}
	}
}

func TestMiddlewareAndRoles(t *testing.T) {
	admin, _, _ := auth.Sign(key, 1, auth.RoleAdmin, time.Hour)
	customer, _, _ := auth.Sign(key, 2, auth.RoleCustomer, time.Hour)

	h := auth.Authenticate(key)(auth.RequireRole(auth.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := auth.FromContext(r.Context())
		json.NewEncoder(w).Encode(map[string]uint{"user": c.UserID})
	})))

	for name, tc := range map[string]struct {
		header string
		want   int
	}{
		"no header":  {"", 401},
		"not bearer": {"Basic abc", 401},
		"bad token":  {"Bearer nope", 401},
		"customer":   {"Bearer " + customer, 403}, // signed in, but the wrong role
		"admin":      {"Bearer " + admin, 200},
	} {
		req := httptest.NewRequest("GET", "/", nil)
		if tc.header != "" {
			req.Header.Set("Authorization", tc.header)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("%s -> %d, want %d", name, rec.Code, tc.want)
		}
	}
}
