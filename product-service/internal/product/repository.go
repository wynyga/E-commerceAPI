package product

import "gorm.io/gorm"

type Repository interface {
	Create(product *Product) error
	FindAll() ([]Product, error)
	FindByID(id uint64) (*Product, error)
	Update(product *Product) error
	Delete(id uint64) error
}
type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(product *Product) error {
	return r.db.Create(product).Error
}

func (r *repository) FindAll() ([]Product, error) {
	var products []Product
	err := r.db.Find(&products).Error
	return products, err
}

func (r *repository) FindByID(id uint64) (*Product, error) {
	var product Product
	err := r.db.First(&product, id).Error
	return &product, err
}
func (r *repository) Update(product *Product) error {
	return r.db.Save(product).Error
}
func (r *repository) Delete(id uint64) error {
	return r.db.Delete(&Product{}, id).Error
}
