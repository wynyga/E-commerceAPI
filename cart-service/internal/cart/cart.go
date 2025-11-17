package cart

import "time"

type Cart struct {
	ID        int        `json:"id" gorm:"primaryKey"`
	UserID    uint       `json:"user_id" gorm:"index"` // Terhubung ke user
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Items     []CartItem `json:"items" gorm:"foreignKey:CartID"`
}

type CartItem struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	CartID    int       `json:"cart_id" gorm:"index"` // Terhubung ke cart
	ProductID uint      `json:"product_id"`           // ID produk
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AddItemRequest struct {
	ProductID uint `json:"product_id" validate:"required"`
	Quantity  int  `json:"quantity" validate:"required,gt=0"`
}
