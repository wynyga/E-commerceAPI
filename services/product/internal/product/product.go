// Package product is the catalogue: the product model and its layers.
package product

import (
	"context"
	"time"
)

type Product struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Price       int64     `gorm:"not null" json:"price"` // whole rupiah
	Stock       int       `gorm:"not null;default:0" json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Input is the body of POST/PUT. Stock may be zero, so it has no `required` tag.
type Input struct {
	Name        string `json:"name" validate:"required,max=255"`
	Description string `json:"description" validate:"max=5000"`
	Price       int64  `json:"price" validate:"gt=0,max=100000000000"`
	Stock       int    `json:"stock" validate:"gte=0,max=1000000"`
}

// QuantityInput is the body of the internal reserve/release calls.
type QuantityInput struct {
	Quantity int `json:"quantity" validate:"gt=0,max=1000"`
}

// Reserved is what a successful reservation tells the order service: the price and name at
// the moment the stock was taken, which the order then keeps as its own snapshot.
type Reserved struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

type Query struct {
	Search string
	Sort   string // name | price | newest
	Limit  int
	Offset int
}

// Repository methods return apperr.NotFound / apperr.Conflict for the matching failures.
type Repository interface {
	List(ctx context.Context, q Query) ([]Product, int, error)
	Get(ctx context.Context, id uint64) (Product, error)
	Create(ctx context.Context, p *Product) error
	Update(ctx context.Context, id uint64, in Input) (Product, error)
	Delete(ctx context.Context, id uint64) error
	// Reserve takes stock with one conditional UPDATE, so two buyers can never both get the last unit.
	Reserve(ctx context.Context, id uint64, qty int) (Reserved, error)
	Release(ctx context.Context, id uint64, qty int) error
}
