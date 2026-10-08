package cart

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/wynyga/ecommerce/shared/apperr"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository { return &repository{db: db} }

var itemNotFound = apperr.NotFound("item not found")

func (r *repository) GetOrCreate(ctx context.Context, userID uint64) (Cart, error) {
	// Insert-if-absent against the unique index: concurrent first requests end up with one cart.
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&Cart{UserID: userID}).Error
	if err != nil {
		return Cart{}, err
	}
	var c Cart
	err = r.db.WithContext(ctx).Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).
		Where("user_id = ?", userID).First(&c).Error
	if c.Items == nil {
		c.Items = []CartItem{}
	}
	return c, err
}

func (r *repository) QuantityOf(ctx context.Context, cartID, productID uint64) (int, error) {
	var item CartItem
	err := r.db.WithContext(ctx).Where("cart_id = ? AND product_id = ?", cartID, productID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return item.Quantity, err
}

func (r *repository) AddQuantity(ctx context.Context, cartID, productID uint64, qty int) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cart_id"}, {Name: "product_id"}},
		DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", qty)}),
	}).Create(&CartItem{CartID: cartID, ProductID: productID, Quantity: qty}).Error
}

// ownedItems limits a query on cart_items to the given user's cart.
func ownedItems(db *gorm.DB, userID uint64) *gorm.DB {
	return db.Where("cart_id IN (?)", db.Session(&gorm.Session{NewDB: true}).Model(&Cart{}).Select("id").Where("user_id = ?", userID))
}

func (r *repository) FindItem(ctx context.Context, userID, itemID uint64) (CartItem, error) {
	var item CartItem
	err := ownedItems(r.db.WithContext(ctx), userID).Where("id = ?", itemID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return CartItem{}, itemNotFound
	}
	return item, err
}

func (r *repository) SetQuantity(ctx context.Context, userID, itemID uint64, qty int) error {
	res := ownedItems(r.db.WithContext(ctx).Model(&CartItem{}), userID).Where("id = ?", itemID).Update("quantity", qty)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return itemNotFound
	}
	return nil
}

func (r *repository) RemoveItem(ctx context.Context, userID, itemID uint64) error {
	// Someone else's item looks exactly like a missing one, so ids cannot be probed.
	res := ownedItems(r.db.WithContext(ctx), userID).Where("id = ?", itemID).Delete(&CartItem{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return itemNotFound
	}
	return nil
}

func (r *repository) Clear(ctx context.Context, userID uint64) error {
	return ownedItems(r.db.WithContext(ctx), userID).Delete(&CartItem{}).Error
}
