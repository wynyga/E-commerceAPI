package cart

import (
	"context"
	"errors"
	"testing"

	"github.com/wynyga/ecommerce/shared/apperr"
)

// ---- fakes ----------------------------------------------------------------

type fakeRepo struct {
	carts map[uint64]*Cart // by user
	next  uint64
}

func newRepo() *fakeRepo { return &fakeRepo{carts: map[uint64]*Cart{}} }

func (f *fakeRepo) GetOrCreate(_ context.Context, userID uint64) (Cart, error) {
	c, ok := f.carts[userID]
	if !ok {
		f.next++
		c = &Cart{ID: f.next, UserID: userID, Items: []CartItem{}}
		f.carts[userID] = c
	}
	return *c, nil
}

func (f *fakeRepo) QuantityOf(_ context.Context, cartID, productID uint64) (int, error) {
	for _, c := range f.carts {
		if c.ID != cartID {
			continue
		}
		for _, it := range c.Items {
			if it.ProductID == productID {
				return it.Quantity, nil
			}
		}
	}
	return 0, nil
}

func (f *fakeRepo) AddQuantity(_ context.Context, cartID, productID uint64, qty int) error {
	for _, c := range f.carts {
		if c.ID != cartID {
			continue
		}
		for i := range c.Items {
			if c.Items[i].ProductID == productID {
				c.Items[i].Quantity += qty
				return nil
			}
		}
		f.next++
		c.Items = append(c.Items, CartItem{ID: f.next, CartID: cartID, ProductID: productID, Quantity: qty})
	}
	return nil
}

func (f *fakeRepo) find(userID, itemID uint64) (*Cart, int) {
	if c, ok := f.carts[userID]; ok {
		for i, it := range c.Items {
			if it.ID == itemID {
				return c, i
			}
		}
	}
	return nil, -1
}

func (f *fakeRepo) FindItem(_ context.Context, userID, itemID uint64) (CartItem, error) {
	if c, i := f.find(userID, itemID); c != nil {
		return c.Items[i], nil
	}
	return CartItem{}, apperr.NotFound("item not found")
}

func (f *fakeRepo) SetQuantity(_ context.Context, userID, itemID uint64, qty int) error {
	c, i := f.find(userID, itemID)
	if c == nil {
		return apperr.NotFound("item not found")
	}
	c.Items[i].Quantity = qty
	return nil
}

func (f *fakeRepo) RemoveItem(_ context.Context, userID, itemID uint64) error {
	c, i := f.find(userID, itemID)
	if c == nil {
		return apperr.NotFound("item not found")
	}
	c.Items = append(c.Items[:i], c.Items[i+1:]...)
	return nil
}

func (f *fakeRepo) Clear(_ context.Context, userID uint64) error {
	if c, ok := f.carts[userID]; ok {
		c.Items = []CartItem{}
	}
	return nil
}

type fakeCatalog map[uint64]Product

func (f fakeCatalog) Get(_ context.Context, id uint64) (Product, error) {
	if p, ok := f[id]; ok {
		return p, nil
	}
	return Product{}, apperr.NotFound("product not found")
}

func setup() (Service, *fakeRepo) {
	repo := newRepo()
	return NewService(repo, fakeCatalog{1: {ID: 1, Name: "Mug", Price: 50_000, Stock: 5}, 2: {ID: 2, Name: "Pen", Price: 5_000, Stock: 500}}), repo
}

var ctx = context.Background()

// ---- tests ----------------------------------------------------------------

func TestAddingTheSameProductTwiceAddsUpOnOneLine(t *testing.T) {
	svc, _ := setup()
	svc.AddItem(ctx, 7, AddInput{ProductID: 1, Quantity: 2})

	c, err := svc.AddItem(ctx, 7, AddInput{ProductID: 1, Quantity: 1})

	if err != nil || len(c.Items) != 1 || c.Items[0].Quantity != 3 {
		t.Fatalf("cart = %+v, err = %v", c, err)
	}
}

func TestAnUnknownProductIsRejected(t *testing.T) {
	svc, _ := setup()

	_, err := svc.AddItem(ctx, 7, AddInput{ProductID: 404, Quantity: 1})

	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
}

func TestQuantityCannotExceedStock(t *testing.T) {
	svc, _ := setup()
	svc.AddItem(ctx, 7, AddInput{ProductID: 1, Quantity: 4})

	_, err := svc.AddItem(ctx, 7, AddInput{ProductID: 1, Quantity: 2}) // 4 + 2 > 5 in stock

	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("err = %v, want conflict", err)
	}
}

func TestQuantityCannotExceedThePerLineCap(t *testing.T) {
	svc, _ := setup()
	svc.AddItem(ctx, 7, AddInput{ProductID: 2, Quantity: 99})

	_, err := svc.AddItem(ctx, 7, AddInput{ProductID: 2, Quantity: 1})

	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("err = %v, want conflict at %d", err, MaxPerItem)
	}
}

func TestUsersCannotSeeOrTouchEachOthersItems(t *testing.T) {
	svc, _ := setup()
	mine, _ := svc.AddItem(ctx, 7, AddInput{ProductID: 1, Quantity: 1})
	itemID := mine.Items[0].ID

	if _, err := svc.RemoveItem(ctx, 8, itemID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("remove err = %v, want not found (never 'forbidden': ids must not be probeable)", err)
	}
	if _, err := svc.SetQuantity(ctx, 8, itemID, 2); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("set err = %v, want not found", err)
	}
	if still, _ := svc.Get(ctx, 7); len(still.Items) != 1 {
		t.Errorf("the owner's cart changed: %+v", still)
	}
}

func TestSetQuantityIsCheckedAgainstStock(t *testing.T) {
	svc, _ := setup()
	c, _ := svc.AddItem(ctx, 7, AddInput{ProductID: 1, Quantity: 1})
	id := c.Items[0].ID

	if _, err := svc.SetQuantity(ctx, 7, id, 6); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("err = %v, want conflict", err)
	}
	got, err := svc.SetQuantity(ctx, 7, id, 5)
	if err != nil || got.Items[0].Quantity != 5 {
		t.Fatalf("cart = %+v, err = %v", got, err)
	}
}

func TestClearEmptiesOnlyTheCallersCart(t *testing.T) {
	svc, _ := setup()
	svc.AddItem(ctx, 7, AddInput{ProductID: 1, Quantity: 1})
	svc.AddItem(ctx, 8, AddInput{ProductID: 1, Quantity: 1})

	if err := svc.Clear(ctx, 7); err != nil {
		t.Fatal(err)
	}

	a, _ := svc.Get(ctx, 7)
	b, _ := svc.Get(ctx, 8)
	if len(a.Items) != 0 || len(b.Items) != 1 {
		t.Errorf("user 7 has %d items, user 8 has %d; want 0 and 1", len(a.Items), len(b.Items))
	}
}
