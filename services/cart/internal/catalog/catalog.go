// Package catalog adapts the product service's HTTP API to the cart's Catalog interface.
package catalog

import (
	"context"
	"net/http"
	"strconv"

	"github.com/wynyga/ecommerce/services/cart/internal/cart"
	"github.com/wynyga/ecommerce/shared/httpclient"
)

type client struct {
	c *httpclient.Client
}

// New returns a Catalog backed by the product service at baseURL. Product reads are public.
func New(baseURL string) cart.Catalog {
	return &client{c: httpclient.New("product", baseURL, "")}
}

func (c *client) Get(ctx context.Context, id uint64) (cart.Product, error) {
	var p cart.Product
	err := c.c.Do(ctx, http.MethodGet, httpclient.Path("/api/v1/products", strconv.FormatUint(id, 10)), "", nil, &p)
	return p, err
}
