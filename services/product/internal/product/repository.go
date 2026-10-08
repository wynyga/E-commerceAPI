package product

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/wynyga/ecommerce/shared/apperr"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository { return &repository{db: db} }

var notFound = apperr.NotFound("product not found")

func (r *repository) List(ctx context.Context, q Query) ([]Product, int, error) {
	tx := r.db.WithContext(ctx).Model(&Product{})
	if term := strings.TrimSpace(q.Search); term != "" {
		like := "%" + escapeLike(term) + "%"
		tx = tx.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	order := "name ASC, id ASC"
	switch q.Sort {
	case "price":
		order = "price ASC, id ASC"
	case "-price":
		order = "price DESC, id ASC"
	case "newest":
		order = "id DESC"
	}

	var items []Product
	err := tx.Order(order).Limit(q.Limit).Offset(q.Offset).Find(&items).Error
	return items, int(total), err
}

func (r *repository) Get(ctx context.Context, id uint64) (Product, error) {
	var p Product
	err := r.db.WithContext(ctx).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Product{}, notFound
	}
	return p, err
}

func (r *repository) Create(ctx context.Context, p *Product) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *repository) Update(ctx context.Context, id uint64, in Input) (Product, error) {
	res := r.db.WithContext(ctx).Model(&Product{}).Where("id = ?", id).
		Updates(map[string]any{"name": in.Name, "description": in.Description, "price": in.Price, "stock": in.Stock})
	if res.Error != nil {
		return Product{}, res.Error
	}
	// RowsAffected is 0 both when the row is missing and when nothing changed, so look it up.
	return r.Get(ctx, id)
}

func (r *repository) Delete(ctx context.Context, id uint64) error {
	res := r.db.WithContext(ctx).Delete(&Product{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return notFound
	}
	return nil
}

func (r *repository) Reserve(ctx context.Context, id uint64, qty int) (Reserved, error) {
	// The stock check is part of the WHERE clause: if another buyer took the last unit
	// first, this matches no row instead of driving stock below zero.
	res := r.db.WithContext(ctx).Exec(
		`UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?`, qty, id, qty)
	if res.Error != nil {
		return Reserved{}, res.Error
	}
	if res.RowsAffected == 0 {
		p, err := r.Get(ctx, id)
		if err != nil {
			return Reserved{}, err
		}
		return Reserved{}, apperr.Conflict("only %d of %q left in stock", p.Stock, p.Name)
	}
	p, err := r.Get(ctx, id)
	if err != nil {
		return Reserved{}, err
	}
	return Reserved{ID: p.ID, Name: p.Name, Price: p.Price}, nil
}

func (r *repository) Release(ctx context.Context, id uint64, qty int) error {
	res := r.db.WithContext(ctx).Exec(`UPDATE products SET stock = stock + ? WHERE id = ?`, qty, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return notFound
	}
	return nil
}

// escapeLike neutralises LIKE wildcards in user input.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
