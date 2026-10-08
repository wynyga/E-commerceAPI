// Package httpclient is how one service calls another. It decodes the shared error body and
// turns remote failures back into apperr categories so callers can handle them uniformly.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wynyga/ecommerce/shared/apperr"
	"github.com/wynyga/ecommerce/shared/httpx"
)

type Client struct {
	name string
	base string
	key  string
	hc   *http.Client
}

// New creates a client for the named service at baseURL. internalKey is sent on every call.
func New(name, baseURL, internalKey string) *Client {
	return &Client{
		name: name,
		base: strings.TrimRight(baseURL, "/"),
		key:  internalKey,
		hc:   &http.Client{Timeout: 5 * time.Second},
	}
}

type errorBody struct {
	Error string `json:"error"`
}

// Do sends in (JSON, may be nil) and decodes the response body into out (may be nil).
// bearer, when set, is forwarded as the caller's own credentials.
func (c *Client) Do(ctx context.Context, method, path, bearer string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set(httpx.HeaderInternalKey, c.key)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return apperr.Unavailable("%s service is unreachable", c.name)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	if resp.StatusCode < 300 {
		if out != nil && len(raw) > 0 {
			return json.Unmarshal(raw, out)
		}
		return nil
	}

	var eb errorBody
	_ = json.Unmarshal(raw, &eb)
	switch resp.StatusCode {
	case http.StatusNotFound:
		return apperr.NotFound("%s", eb.Error)
	case http.StatusConflict:
		return apperr.Conflict("%s", eb.Error)
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return apperr.Invalid("%s", eb.Error)
	case http.StatusUnauthorized:
		return apperr.Unauthorized("%s", eb.Error)
	case http.StatusForbidden:
		return apperr.Forbidden("%s", eb.Error)
	default:
		return apperr.Unavailable("%s service failed (%d)", c.name, resp.StatusCode)
	}
}

// Path joins escaped segments, e.g. Path("/internal/products", id, "reserve").
func Path(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		if strings.HasPrefix(p, "/") {
			b.WriteString(p)
			continue
		}
		b.WriteString("/")
		b.WriteString(url.PathEscape(p))
	}
	return b.String()
}
