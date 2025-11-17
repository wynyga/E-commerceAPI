package cart

import (
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// Service mendefinisikan interface untuk logika bisnis keranjang.
type Service interface {
	// Mendapatkan keranjang pengguna (membuat jika belum ada).
	GetOrCreateCart(userID uint) (*Cart, error)

	// Menambahkan item ke keranjang.
	AddItemToCart(userID uint, req AddItemRequest) (*Cart, error)

	// Menghapus item dari keranjang.
	RemoveItemFromCart(userID uint, cartItemID int) (*Cart, error)
}

type service struct {
	repo Repository
}

// NewService membuat instance service baru.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Implementasi fungsi-fungsi interface

// GetOrCreateCart adalah logika bisnis utama:
// 1. Coba cari keranjang user.
// 2. Jika tidak ada (gorm.ErrRecordNotFound), BUAT keranjang baru.
// 3. Kembalikan keranjang tersebut.
func (s *service) GetOrCreateCart(userID uint) (*Cart, error) {
	// 1. Coba cari
	cart, err := s.repo.GetCartByUserID(userID)

	if err != nil {
		// 2. Jika tidak ada, buat baru
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("No cart found for user %d, creating one.", userID)
			newCart := &Cart{
				UserID: userID,
				Items:  []CartItem{}, // Inisialisasi slice kosong
			}

			if err := s.repo.CreateCart(newCart); err != nil {
				return nil, fmt.Errorf("failed to create cart: %w", err)
			}
			// Kembalikan keranjang yang baru dibuat
			return newCart, nil
		}

		// Error lain selain "tidak ditemukan"
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	// 3. Jika ditemukan, kembalikan
	return cart, nil
}

// AddItemToCart menambahkan item ke keranjang.
// 1. Dapatkan keranjang user (atau buat baru)
// 2. Cek apakah item (product_id) sudah ada di keranjang
// 3. Jika sudah ada, tambahkan kuantitasnya
// 4. Jika belum ada, buat item baru
// 5. Kembalikan kondisi keranjang terbaru
func (s *service) AddItemToCart(userID uint, req AddItemRequest) (*Cart, error) {
	// 1. Dapatkan keranjang
	cart, err := s.GetOrCreateCart(userID)
	if err != nil {
		return nil, err
	}

	// 2. Cek apakah item sudah ada
	existingItem, err := s.repo.FindItemByCartIdAndProductID(cart.ID, req.ProductID)

	if err != nil {
		// 3. Jika belum ada (error not found)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Item %d not found in cart %d, creating new entry.", req.ProductID, cart.ID)
			newItem := &CartItem{
				CartID:    cart.ID,
				ProductID: req.ProductID,
				Quantity:  req.Quantity,
			}
			if err := s.repo.CreateCartItem(newItem); err != nil {
				return nil, fmt.Errorf("failed to create cart item: %w", err)
			}
		} else {
			// Error lain
			return nil, fmt.Errorf("failed to find cart item: %w", err)
		}
	} else {
		// 4. Jika sudah ada, update kuantitas
		log.Printf("Item %d found in cart %d, updating quantity.", req.ProductID, cart.ID)
		existingItem.Quantity += req.Quantity
		if err := s.repo.UpdateCartItem(existingItem); err != nil {
			return nil, fmt.Errorf("failed to update cart item: %w", err)
		}
	}

	// 5. Kembalikan keranjang versi terbaru (dengan item yang sudah di-preload)
	updatedCart, err := s.repo.GetCartByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated cart: %w", err)
	}
	return updatedCart, nil
}

// RemoveItemFromCart menghapus item dari keranjang.
// 1. Dapatkan keranjang user (untuk verifikasi kepemilikan)
// 2. Cari item yang ingin dihapus (berdasarkan cartItemID)
// 3. PASTIKAN item.CartID == cart.ID (verifikasi kepemilikan)
// 4. Hapus item
// 5. Kembalikan kondisi keranjang terbaru
func (s *service) RemoveItemFromCart(userID uint, cartItemID int) (*Cart, error) {
	// 1. Dapatkan keranjang user
	cart, err := s.repo.GetCartByUserID(userID)
	if err != nil {
		// Jika user bahkan tidak punya keranjang, dia pasti tidak punya item
		return nil, fmt.Errorf("cart not found: %w", err)
	}

	// 2. Cari item
	item, err := s.repo.GetCartItemByID(cartItemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("item not found")
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	// 3. Verifikasi Kepemilikan (SANGAT PENTING!)
	// Mencegah user A menghapus item milik user B
	if item.CartID != cart.ID {
		log.Printf("SECURITY: User %d tried to delete item %d belonging to cart %d", userID, item.ID, item.CartID)
		return nil, fmt.Errorf("unauthorized: item does not belong to user")
	}

	// 4. Hapus item
	if err := s.repo.DeleteCartItem(item); err != nil {
		return nil, fmt.Errorf("failed to delete item: %w", err)
	}

	// 5. Kembalikan keranjang versi terbaru
	updatedCart, err := s.repo.GetCartByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated cart: %w", err)
	}
	return updatedCart, nil
}
