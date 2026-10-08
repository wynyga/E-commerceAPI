// Package clients adapts the product and cart services' HTTP APIs to the interfaces the
// order domain depends on.
package clients

import (
	"context"
	"net/http"
	"strconv"

	"github.com/wynyga/ecommerce/services/order/internal/order"
	"github.com/wynyga/ecommerce/shared/httpclient"
)

type catalog struct {
	c *httpclient.Client
}

// NewCatalog talks to the product service's internal stock endpoints.
func NewCatalog(baseURL, internalKey string) order.Catalog {
	return &catalog{c: httpclient.New("product", baseURL, internalKey)}
}

func (p *catalog) Reserve(ctx context.Context, productID uint64, qty int) (order.Reserved, error) {
	var out order.Reserved
	path := httpclient.Path("/internal/products", strconv.FormatUint(productID, 10), "reserve")
	err := p.c.Do(ctx, http.MethodPost, path, "", map[string]int{"quantity": qty}, &out)
	return out, err
}

func (p *catalog) Release(ctx context.Context, productID uint64, qty int) error {
	path := httpclient.Path("/internal/products", strconv.FormatUint(productID, 10), "release")
	return p.c.Do(ctx, http.MethodPost, path, "", map[string]int{"quantity": qty}, nil)
}

type carts struct {
	c *httpclient.Client
}

// NewCarts talks to the cart service on the buyer's behalf, using the buyer's own token.
func NewCarts(baseURL string) order.Carts {
	return &carts{c: httpclient.New("cart", baseURL, "")}
}

func (c *carts) Get(ctx context.Context, bearer string) (order.CartView, error) {
	var out order.CartView
	err := c.c.Do(ctx, http.MethodGet, "/api/v1/cart", bearer, nil, &out)
	return out, err
}

func (c *carts) Clear(ctx context.Context, bearer string) error {
	return c.c.Do(ctx, http.MethodDelete, "/api/v1/cart", bearer, nil, nil)
}
