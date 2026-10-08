package order

import (
	"context"
	"errors"
	"log"
	"sort"
	"time"

	"github.com/wynyga/ecommerce/shared/apperr"
)

type service struct {
	repo    Repository
	catalog Catalog
	carts   Carts
}

func NewService(repo Repository, catalog Catalog, carts Carts) Service {
	return &service{repo: repo, catalog: catalog, carts: carts}
}

type taken struct {
	productID uint64
	qty       int
}

// Checkout reserves stock line by line, then records the order. If anything fails after
// stock has been taken, the stock is given back, so a failed checkout never leaves units
// locked up with no order behind them.
//
// With an idempotency key, repeating the call (a retry after a timeout, a double click)
// returns the original order instead of charging twice.
func (s *service) Checkout(ctx context.Context, userID uint64, bearer, key string) (Order, bool, error) {
	if len(key) > 100 {
		return Order{}, false, apperr.Invalid("Idempotency-Key must be at most 100 characters")
	}
	if key != "" {
		if existing, err := s.repo.FindByKey(ctx, userID, key); err == nil {
			return existing, false, nil
		} else if !errors.Is(err, apperr.ErrNotFound) {
			return Order{}, false, err
		}
	}

	cart, err := s.carts.Get(ctx, bearer)
	if err != nil {
		return Order{}, false, err
	}
	if len(cart.Items) == 0 {
		// A parallel request with the same key may have just created the order and emptied
		// the cart; that is a replay, not an empty cart.
		if key != "" {
			if existing, err := s.repo.FindByKey(ctx, userID, key); err == nil {
				return existing, false, nil
			}
		}
		return Order{}, false, apperr.Invalid("your cart is empty")
	}
	// A stable order makes behaviour predictable and keeps the reserve calls of two buyers
	// touching rows in the same sequence.
	lines := append([]CartLine(nil), cart.Items...)
	sort.Slice(lines, func(i, j int) bool { return lines[i].ProductID < lines[j].ProductID })

	var reserved []taken
	items := make([]OrderItem, 0, len(lines))
	var total int64
	for _, l := range lines {
		r, err := s.catalog.Reserve(ctx, l.ProductID, l.Quantity)
		if err != nil {
			s.release(reserved)
			if errors.Is(err, apperr.ErrNotFound) {
				return Order{}, false, apperr.Conflict("product %d is no longer available", l.ProductID)
			}
			return Order{}, false, err
		}
		reserved = append(reserved, taken{l.ProductID, l.Quantity})
		items = append(items, OrderItem{ProductID: l.ProductID, Name: r.Name, UnitPrice: r.Price, Quantity: l.Quantity})
		total += r.Price * int64(l.Quantity)
	}

	o := Order{UserID: userID, Status: StatusPending, Total: total, Items: items}
	if key != "" {
		o.IdempotencyKey = &key
	}
	if err := s.repo.Create(ctx, &o); err != nil {
		s.release(reserved)
		if errors.Is(err, ErrDuplicateKey) {
			// A parallel request with the same key won; give back what this one took and return its order.
			existing, ferr := s.repo.FindByKey(ctx, userID, key)
			return existing, false, ferr
		}
		return Order{}, false, err
	}

	// The order stands even if emptying the cart fails; the buyer can clear it themselves.
	if err := s.carts.Clear(ctx, bearer); err != nil {
		log.Printf("order %d created but the cart could not be cleared: %v", o.ID, err)
	}
	return o, true, nil
}

// Cancel marks the order cancelled first (a conditional update, so only one caller ever
// gets to return the stock), then returns each item's stock. If returning an item fails,
// the order is reopened and the items already returned are reserved again, so the order
// and the catalogue still agree and the caller can simply retry.
func (s *service) Cancel(ctx context.Context, userID, id uint64) (Order, error) {
	o, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		return Order{}, err
	}
	if err := s.repo.MarkCancelled(ctx, userID, id); err != nil {
		return Order{}, err
	}

	var returned []taken
	for _, it := range o.Items {
		if err := s.catalog.Release(ctx, it.ProductID, it.Quantity); err != nil {
			s.reopen(id)
			s.retake(returned)
			return Order{}, err
		}
		returned = append(returned, taken{it.ProductID, it.Quantity})
	}
	o.Status = StatusCancelled
	return o, nil
}

func (s *service) Get(ctx context.Context, userID, id uint64) (Order, error) {
	return s.repo.Get(ctx, userID, id)
}

func (s *service) List(ctx context.Context, userID uint64, limit, offset int) ([]Order, int, error) {
	return s.repo.List(ctx, userID, limit, offset)
}

// release hands units back to the catalogue on a fresh context: the request that triggered
// it may already be cancelled, and leaving stock locked is worse than waiting a moment.
func (s *service) release(items []taken) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 5*time.Second)
	defer cancel()
	for _, t := range items {
		if err := s.catalog.Release(ctx, t.productID, t.qty); err != nil {
			log.Printf("CRITICAL: could not release %d x product %d: %v - stock needs manual reconciliation", t.qty, t.productID, err)
		}
	}
}

func (s *service) reopen(id uint64) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 5*time.Second)
	defer cancel()
	if err := s.repo.Reopen(ctx, id); err != nil {
		log.Printf("CRITICAL: could not reopen order %d: %v - manual reconciliation needed", id, err)
	}
}

// retake reserves again the units a failed cancel had already returned.
func (s *service) retake(items []taken) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 5*time.Second)
	defer cancel()
	for _, t := range items {
		if _, err := s.catalog.Reserve(ctx, t.productID, t.qty); err != nil {
			log.Printf("CRITICAL: could not re-reserve %d x product %d: %v - stock needs manual reconciliation", t.qty, t.productID, err)
		}
	}
}
