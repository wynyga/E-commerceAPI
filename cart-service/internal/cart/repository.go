package cart

import (
	"gorm.io/gorm"
)

type Repository interface {
	//Mencari keranjang berdasarkan userID
	GetCartByUserID(userID uint) (*Cart, error)
	//Membuat keranjang baru
	CreateCart(cart *Cart) error
	//Mencari item spesifik di dalam keranjang
	FindItemByCartIdAndProductID(cartID int, productID uint) (*CartItem, error)
	//Mencari item berdasarkan ID unik (primary key)
	GetCartItemByID(itemID int) (*CartItem, error)
	//Menambahkan item ke keranjang
	CreateCartItem(item *CartItem) error
	//Memperbarui item yang sudah ada di keranjang
	UpdateCartItem(item *CartItem) error
	//Menghapus item dari keranjang
	DeleteCartItem(item *CartItem) error
	//Mendapatkan semua item dalam keranjang tertentu
	GetCartItems(cartID int) ([]CartItem, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Implementasi fungsi interface

func (r *repository) GetCartByUserID(userID uint) (*Cart, error) {
	var cart Cart
	if err := r.db.Preload("Items").Where("user_id = ?", userID).First(&cart).Error; err != nil {
		return nil, err
	}
	return &cart, nil
}

func (r *repository) CreateCart(cart *Cart) error {
	return r.db.Create(cart).Error
}

func (r *repository) FindItemByCartIdAndProductID(cartID int, productID uint) (*CartItem, error) {
	var item CartItem
	if err := r.db.Where("cart_id = ? AND product_id = ?", cartID, productID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *repository) GetCartItemByID(itemID int) (*CartItem, error) {
	var item CartItem
	if err := r.db.First(&item, itemID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *repository) CreateCartItem(item *CartItem) error {
	return r.db.Create(item).Error
}

func (r *repository) UpdateCartItem(item *CartItem) error {
	// GORM Save() akan mengupdate semua field jika struct memiliki Primary Key.
	// Ini termasuk kuantitas dan updated_at.
	return r.db.Save(item).Error
}

func (r *repository) DeleteCartItem(item *CartItem) error {
	return r.db.Delete(item).Error
}

func (r *repository) GetCartItems(cartID int) ([]CartItem, error) {
	var items []CartItem
	if err := r.db.Where("cart_id = ?", cartID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
