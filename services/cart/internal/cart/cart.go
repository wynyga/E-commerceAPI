// Package cart is the shopping cart: one cart per user, one line per product.
package cart

import (
	"context"
	"time"
)

// MaxPerItem caps how many units of one product a cart may hold.
const MaxPerItem = 99

type Cart struct {
	ID        uint64     `gorm:"primaryKey" json:"id"`
	UserID    uint64     `gorm:"uniqueIndex" json:"user_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Items     []CartItem `gorm:"foreignKey:CartID" json:"items"`
}

type CartItem struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	CartID    uint64    `json:"cart_id"`
	ProductID uint64    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AddInput struct {
	ProductID uint64 `json:"product_id" validate:"required,gt=0"`
	Quantity  int    `json:"quantity" validate:"gt=0,max=99"`
}

type QuantityInput struct {
	Quantity int `json:"quantity" validate:"gt=0,max=99"`
}

// Product is the slice of the catalogue the cart needs to validate a line.
type Product struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Price int64  `json:"price"`
	Stock int    `json:"stock"`
}

// Catalog is the cart's view of the product service.
type Catalog interface {
	// Get returns apperr.NotFound for an unknown product.
	Get(ctx context.Context, id uint64) (Product, error)
}

// Repository methods are all scoped to the owning user, so one user can never touch another's cart.
type Repository interface {
	GetOrCreate(ctx context.Context, userID uint64) (Cart, error)
	QuantityOf(ctx context.Context, cartID, productID uint64) (int, error)
	// AddQuantity inserts the line or adds to it in one statement.
	AddQuantity(ctx context.Context, cartID, productID uint64, qty int) error
	// FindItem returns apperr.NotFound unless the item is in this user's cart.
	FindItem(ctx context.Context, userID, itemID uint64) (CartItem, error)
	SetQuantity(ctx context.Context, userID, itemID uint64, qty int) error
	RemoveItem(ctx context.Context, userID, itemID uint64) error
	Clear(ctx context.Context, userID uint64) error
}

type Service interface {
	Get(ctx context.Context, userID uint64) (Cart, error)
	AddItem(ctx context.Context, userID uint64, in AddInput) (Cart, error)
	SetQuantity(ctx context.Context, userID, itemID uint64, qty int) (Cart, error)
	RemoveItem(ctx context.Context, userID, itemID uint64) (Cart, error)
	Clear(ctx context.Context, userID uint64) error
}
