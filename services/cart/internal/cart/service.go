package cart

import (
	"context"

	"github.com/wynyga/ecommerce/shared/apperr"
)

type service struct {
	repo    Repository
	catalog Catalog
}

func NewService(repo Repository, catalog Catalog) Service {
	return &service{repo: repo, catalog: catalog}
}

func (s *service) Get(ctx context.Context, userID uint64) (Cart, error) {
	return s.repo.GetOrCreate(ctx, userID)
}

// checkQuantity enforces the per-line cap and the stock currently on the shelf. Stock is
// only a soft check here: the order service reserves it for real at checkout.
func checkQuantity(p Product, want int) error {
	if want > MaxPerItem {
		return apperr.Conflict("a cart can hold at most %d of one product", MaxPerItem)
	}
	if want > p.Stock {
		return apperr.Conflict("only %d of %q in stock", p.Stock, p.Name)
	}
	return nil
}

func (s *service) AddItem(ctx context.Context, userID uint64, in AddInput) (Cart, error) {
	p, err := s.catalog.Get(ctx, in.ProductID)
	if err != nil {
		return Cart{}, err
	}
	c, err := s.repo.GetOrCreate(ctx, userID)
	if err != nil {
		return Cart{}, err
	}
	have, err := s.repo.QuantityOf(ctx, c.ID, in.ProductID)
	if err != nil {
		return Cart{}, err
	}
	if err := checkQuantity(p, have+in.Quantity); err != nil {
		return Cart{}, err
	}
	if err := s.repo.AddQuantity(ctx, c.ID, in.ProductID, in.Quantity); err != nil {
		return Cart{}, err
	}
	return s.repo.GetOrCreate(ctx, userID)
}

func (s *service) SetQuantity(ctx context.Context, userID, itemID uint64, qty int) (Cart, error) {
	item, err := s.repo.FindItem(ctx, userID, itemID)
	if err != nil {
		return Cart{}, err
	}
	p, err := s.catalog.Get(ctx, item.ProductID)
	if err != nil {
		return Cart{}, err
	}
	if err := checkQuantity(p, qty); err != nil {
		return Cart{}, err
	}
	if err := s.repo.SetQuantity(ctx, userID, itemID, qty); err != nil {
		return Cart{}, err
	}
	return s.repo.GetOrCreate(ctx, userID)
}

func (s *service) RemoveItem(ctx context.Context, userID, itemID uint64) (Cart, error) {
	if err := s.repo.RemoveItem(ctx, userID, itemID); err != nil {
		return Cart{}, err
	}
	return s.repo.GetOrCreate(ctx, userID)
}

func (s *service) Clear(ctx context.Context, userID uint64) error {
	return s.repo.Clear(ctx, userID)
}
