// Package order turns a cart into an order. The hard part is that it spans three services:
// stock lives in the catalogue, the cart in the cart service, the order here, and there is no
// shared transaction between them. The service therefore orders its steps so that any failure
// can be undone (see Checkout and Cancel).
package order

import (
	"context"
	"errors"
	"time"
)

const (
	StatusPending   = "pending"
	StatusCancelled = "cancelled"
)

type Order struct {
	ID             uint64      `gorm:"primaryKey" json:"id"`
	UserID         uint64      `json:"user_id"`
	Status         string      `json:"status"`
	Total          int64       `json:"total"` // whole rupiah
	IdempotencyKey *string     `json:"-"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	Items          []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
}

type OrderItem struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	OrderID   uint64 `json:"-"`
	ProductID uint64 `json:"product_id"`
	Name      string `json:"name"`
	UnitPrice int64  `json:"unit_price"`
	Quantity  int    `json:"quantity"`
}

// ErrDuplicateKey is what Repository.Create returns when the (user, idempotency key) pair
// already has an order.
var ErrDuplicateKey = errors.New("duplicate idempotency key")

// Reserved is what the catalogue answers when stock is taken: the name and price at that moment.
type Reserved struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

type CartLine struct {
	ProductID uint64 `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type CartView struct {
	Items []CartLine `json:"items"`
}

// Catalog is the order service's view of the product service.
type Catalog interface {
	// Reserve takes qty units or fails (Conflict when short, NotFound when the product is gone).
	Reserve(ctx context.Context, productID uint64, qty int) (Reserved, error)
	Release(ctx context.Context, productID uint64, qty int) error
}

// Carts is the order service's view of the cart service, acting with the buyer's own token.
type Carts interface {
	Get(ctx context.Context, bearer string) (CartView, error)
	Clear(ctx context.Context, bearer string) error
}

// Repository methods return apperr.NotFound / apperr.Conflict for the matching failures.
type Repository interface {
	// Create stores the order and its items in one transaction; ErrDuplicateKey on a repeated key.
	Create(ctx context.Context, o *Order) error
	FindByKey(ctx context.Context, userID uint64, key string) (Order, error)
	Get(ctx context.Context, userID, id uint64) (Order, error)
	List(ctx context.Context, userID uint64, limit, offset int) ([]Order, int, error)
	// MarkCancelled flips pending to cancelled with a conditional update (Conflict if it was not pending).
	MarkCancelled(ctx context.Context, userID, id uint64) error
	// Reopen undoes MarkCancelled. Used only to compensate a failed stock release.
	Reopen(ctx context.Context, id uint64) error
}

type Service interface {
	// Checkout returns the order and whether it was newly created (false for an idempotent replay).
	Checkout(ctx context.Context, userID uint64, bearer, idempotencyKey string) (Order, bool, error)
	Cancel(ctx context.Context, userID, id uint64) (Order, error)
	Get(ctx context.Context, userID, id uint64) (Order, error)
	List(ctx context.Context, userID uint64, limit, offset int) ([]Order, int, error)
}
