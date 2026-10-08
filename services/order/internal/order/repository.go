package order

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/wynyga/ecommerce/shared/apperr"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository { return &repository{db: db} }

var notFound = apperr.NotFound("order not found")

// withItems loads items in a stable order.
func withItems(db *gorm.DB) *gorm.DB {
	return db.Preload("Items", func(tx *gorm.DB) *gorm.DB { return tx.Order("id ASC") })
}

func (r *repository) Create(ctx context.Context, o *Order) error {
	// Order and items are written together or not at all.
	err := r.db.WithContext(ctx).Create(o).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrDuplicateKey
	}
	return err
}

func (r *repository) FindByKey(ctx context.Context, userID uint64, key string) (Order, error) {
	var o Order
	err := withItems(r.db.WithContext(ctx)).Where("user_id = ? AND idempotency_key = ?", userID, key).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Order{}, notFound
	}
	return o, err
}

// Get is scoped to the owner: someone else's order looks exactly like a missing one.
func (r *repository) Get(ctx context.Context, userID, id uint64) (Order, error) {
	var o Order
	err := withItems(r.db.WithContext(ctx)).Where("user_id = ? AND id = ?", userID, id).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Order{}, notFound
	}
	return o, err
}

func (r *repository) List(ctx context.Context, userID uint64, limit, offset int) ([]Order, int, error) {
	tx := r.db.WithContext(ctx).Model(&Order{}).Where("user_id = ?", userID)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var orders []Order
	err := withItems(tx).Order("id DESC").Limit(limit).Offset(offset).Find(&orders).Error
	return orders, int(total), err
}

func (r *repository) MarkCancelled(ctx context.Context, userID, id uint64) error {
	res := r.db.WithContext(ctx).Model(&Order{}).
		Where("id = ? AND user_id = ? AND status = ?", id, userID, StatusPending).
		Update("status", StatusCancelled)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperr.Conflict("only a pending order can be cancelled")
	}
	return nil
}

func (r *repository) Reopen(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&Order{}).Where("id = ?", id).Update("status", StatusPending).Error
}
