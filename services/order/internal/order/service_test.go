package order

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/wynyga/ecommerce/shared/apperr"
)

// ---- fakes ----------------------------------------------------------------

type fakeCatalog struct {
	mu         sync.Mutex
	stock      map[uint64]int
	price      map[uint64]int64
	missing    map[uint64]bool
	releaseErr map[uint64]error
}

func newCatalog() *fakeCatalog {
	return &fakeCatalog{
		stock:      map[uint64]int{1: 10, 2: 5, 3: 1},
		price:      map[uint64]int64{1: 50_000, 2: 120_000, 3: 999_000},
		missing:    map[uint64]bool{},
		releaseErr: map[uint64]error{},
	}
}

func (f *fakeCatalog) Reserve(_ context.Context, id uint64, qty int) (Reserved, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.missing[id] {
		return Reserved{}, apperr.NotFound("product not found")
	}
	if f.stock[id] < qty {
		return Reserved{}, apperr.Conflict("only %d left", f.stock[id])
	}
	f.stock[id] -= qty
	return Reserved{ID: id, Name: "P" + string(rune('0'+id)), Price: f.price[id]}, nil
}

func (f *fakeCatalog) Release(_ context.Context, id uint64, qty int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.releaseErr[id]; err != nil {
		return err
	}
	f.stock[id] += qty
	return nil
}

type fakeCarts struct {
	items    []CartLine
	clearErr error
	cleared  bool
	onGet    func() // lets a test change the world while a checkout is in flight
}

func (f *fakeCarts) Get(context.Context, string) (CartView, error) {
	if f.onGet != nil {
		f.onGet()
	}
	return CartView{Items: f.items}, nil
}
func (f *fakeCarts) Clear(context.Context, string) error {
	f.cleared = true
	return f.clearErr
}

type fakeRepo struct {
	orders    map[uint64]*Order
	next      uint64
	createErr error
	// raceWinner, when set, is stored under the key and Create then reports a duplicate,
	// as if a parallel request with the same Idempotency-Key had committed first.
	raceWinner *Order
	reopened   []uint64
}

func newRepo() *fakeRepo { return &fakeRepo{orders: map[uint64]*Order{}} }

func (f *fakeRepo) Create(_ context.Context, o *Order) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.raceWinner != nil {
		f.next++
		w := *f.raceWinner
		w.ID, w.UserID, w.IdempotencyKey = f.next, o.UserID, o.IdempotencyKey
		f.orders[w.ID] = &w
		return ErrDuplicateKey
	}
	f.next++
	o.ID = f.next
	cp := *o
	f.orders[o.ID] = &cp
	return nil
}

func (f *fakeRepo) FindByKey(_ context.Context, userID uint64, key string) (Order, error) {
	for _, o := range f.orders {
		if o.UserID == userID && o.IdempotencyKey != nil && *o.IdempotencyKey == key {
			return *o, nil
		}
	}
	return Order{}, apperr.NotFound("order not found")
}

func (f *fakeRepo) Get(_ context.Context, userID, id uint64) (Order, error) {
	if o, ok := f.orders[id]; ok && o.UserID == userID {
		return *o, nil
	}
	return Order{}, apperr.NotFound("order not found")
}

func (f *fakeRepo) List(context.Context, uint64, int, int) ([]Order, int, error) { return nil, 0, nil }

func (f *fakeRepo) MarkCancelled(_ context.Context, userID, id uint64) error {
	o, ok := f.orders[id]
	if !ok || o.UserID != userID || o.Status != StatusPending {
		return apperr.Conflict("only a pending order can be cancelled")
	}
	o.Status = StatusCancelled
	return nil
}

func (f *fakeRepo) Reopen(_ context.Context, id uint64) error {
	f.orders[id].Status = StatusPending
	f.reopened = append(f.reopened, id)
	return nil
}

func setup(lines ...CartLine) (Service, *fakeCatalog, *fakeCarts, *fakeRepo) {
	cat, carts, repo := newCatalog(), &fakeCarts{items: lines}, newRepo()
	return NewService(repo, cat, carts), cat, carts, repo
}

var ctx = context.Background()

// ---- checkout -------------------------------------------------------------

func TestCheckoutTakesStockSnapshotsPricesAndClearsTheCart(t *testing.T) {
	svc, cat, carts, _ := setup(CartLine{1, 2}, CartLine{2, 1})

	o, created, err := svc.Checkout(ctx, 7, "tok", "")
	if err != nil || !created {
		t.Fatalf("Checkout = %v, created=%v", err, created)
	}
	if want := int64(2*50_000 + 120_000); o.Total != want {
		t.Errorf("total = %d, want %d", o.Total, want)
	}
	if o.Status != StatusPending || len(o.Items) != 2 || o.Items[0].UnitPrice != 50_000 {
		t.Errorf("order = %+v", o)
	}
	if cat.stock[1] != 8 || cat.stock[2] != 4 {
		t.Errorf("stock = %v, want 1:8 2:4", cat.stock)
	}
	if !carts.cleared {
		t.Error("cart should be cleared after a successful checkout")
	}
}

func TestOrderKeepsTheReservedPriceWhenTheCatalogChangesLater(t *testing.T) {
	svc, cat, _, _ := setup(CartLine{1, 1})
	o, _, _ := svc.Checkout(ctx, 7, "tok", "")

	cat.price[1] = 1_000_000 // repriced afterwards

	got, _ := svc.Get(ctx, 7, o.ID)
	if got.Total != 50_000 || got.Items[0].UnitPrice != 50_000 {
		t.Errorf("order changed with the catalogue: %+v", got)
	}
}

func TestEmptyCartIsRejectedAndNothingIsReserved(t *testing.T) {
	svc, cat, _, _ := setup()

	_, _, err := svc.Checkout(ctx, 7, "tok", "")

	if !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("err = %v, want invalid", err)
	}
	if cat.stock[1] != 10 {
		t.Errorf("stock touched: %v", cat.stock)
	}
}

// The second line cannot be satisfied: the first line's stock must come back.
func TestShortStockOnALaterLineReleasesTheEarlierOnes(t *testing.T) {
	svc, cat, _, repo := setup(CartLine{1, 3}, CartLine{3, 2}) // product 3 has only 1

	_, _, err := svc.Checkout(ctx, 7, "tok", "")

	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("err = %v, want conflict", err)
	}
	if cat.stock[1] != 10 || cat.stock[3] != 1 {
		t.Errorf("stock = %v, want everything restored", cat.stock)
	}
	if len(repo.orders) != 0 {
		t.Error("no order may exist after a failed checkout")
	}
}

func TestAProductRemovedFromTheCatalogueIsAConflictAndStockIsRestored(t *testing.T) {
	svc, cat, _, _ := setup(CartLine{1, 1}, CartLine{2, 1})
	cat.missing[2] = true

	_, _, err := svc.Checkout(ctx, 7, "tok", "")

	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("err = %v, want conflict", err)
	}
	if cat.stock[1] != 10 {
		t.Errorf("stock = %v", cat.stock)
	}
}

func TestStockIsReleasedWhenSavingTheOrderFails(t *testing.T) {
	svc, cat, carts, repo := setup(CartLine{1, 2}, CartLine{2, 2})
	repo.createErr = errors.New("db down")

	if _, _, err := svc.Checkout(ctx, 7, "tok", ""); err == nil {
		t.Fatal("want error")
	}
	if cat.stock[1] != 10 || cat.stock[2] != 5 {
		t.Errorf("stock = %v, want everything restored", cat.stock)
	}
	if carts.cleared {
		t.Error("the cart must be kept when the order was not created")
	}
}

func TestACartThatCannotBeClearedDoesNotFailTheCheckout(t *testing.T) {
	svc, _, carts, _ := setup(CartLine{1, 1})
	carts.clearErr = apperr.Unavailable("cart service is unreachable")

	_, created, err := svc.Checkout(ctx, 7, "tok", "")

	if err != nil || !created {
		t.Fatalf("Checkout = %v, created=%v; the order exists, so it must succeed", err, created)
	}
}

// ---- idempotency ----------------------------------------------------------

func TestRepeatingACheckoutWithTheSameKeyReturnsTheFirstOrder(t *testing.T) {
	svc, cat, _, repo := setup(CartLine{1, 2})
	first, created, _ := svc.Checkout(ctx, 7, "tok", "key-1")
	if !created {
		t.Fatal("first call must create")
	}
	stockAfterFirst := cat.stock[1]

	again, created, err := svc.Checkout(ctx, 7, "tok", "key-1")

	if err != nil || created || again.ID != first.ID {
		t.Fatalf("replay = %+v created=%v err=%v", again, created, err)
	}
	if cat.stock[1] != stockAfterFirst || len(repo.orders) != 1 {
		t.Errorf("replay took stock or made a second order: stock=%v orders=%d", cat.stock, len(repo.orders))
	}
}

func TestTheSameKeyFromAnotherUserIsNotAReplay(t *testing.T) {
	svc, _, _, repo := setup(CartLine{1, 1})
	svc.Checkout(ctx, 7, "tok", "shared-key")

	_, created, _ := svc.Checkout(ctx, 8, "tok2", "shared-key")

	if !created || len(repo.orders) != 2 {
		t.Errorf("created=%v orders=%d, want a separate order per user", created, len(repo.orders))
	}
}

// Two parallel requests with one key both reserve stock; the loser must hand its stock back.
func TestALostIdempotencyRaceReleasesStockAndReturnsTheWinner(t *testing.T) {
	svc, cat, _, repo := setup(CartLine{1, 4})
	repo.raceWinner = &Order{Status: StatusPending, Total: 200_000}

	o, created, err := svc.Checkout(ctx, 7, "tok", "key-race")

	if err != nil || created {
		t.Fatalf("Checkout = %v, created=%v", err, created)
	}
	if o.Total != 200_000 {
		t.Errorf("returned %+v, want the winner's order", o)
	}
	if cat.stock[1] != 10 {
		t.Errorf("stock = %d, the loser's reservation must be released", cat.stock[1])
	}
}

func TestOverlongIdempotencyKeyIsRejected(t *testing.T) {
	svc, _, _, _ := setup(CartLine{1, 1})
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'a'
	}
	if _, _, err := svc.Checkout(ctx, 7, "tok", string(long)); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("err = %v, want invalid", err)
	}
}

// ---- cancel ---------------------------------------------------------------

func TestCancelReturnsStockOnce(t *testing.T) {
	svc, cat, _, _ := setup(CartLine{1, 3}, CartLine{2, 2})
	o, _, _ := svc.Checkout(ctx, 7, "tok", "")

	got, err := svc.Cancel(ctx, 7, o.ID)
	if err != nil || got.Status != StatusCancelled {
		t.Fatalf("Cancel = %+v, %v", got, err)
	}
	if cat.stock[1] != 10 || cat.stock[2] != 5 {
		t.Errorf("stock = %v, want fully returned", cat.stock)
	}

	if _, err := svc.Cancel(ctx, 7, o.ID); !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("second cancel err = %v, want conflict", err)
	}
	if cat.stock[1] != 10 {
		t.Errorf("stock = %v: a second cancel must not return stock again", cat.stock)
	}
}

func TestSomeoneElsesOrderLooksMissing(t *testing.T) {
	svc, cat, _, _ := setup(CartLine{1, 1})
	o, _, _ := svc.Checkout(ctx, 7, "tok", "")

	if _, err := svc.Cancel(ctx, 99, o.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
	if _, err := svc.Get(ctx, 99, o.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("Get err = %v, want not found", err)
	}
	if cat.stock[1] != 9 {
		t.Errorf("stock = %v: another user's cancel must change nothing", cat.stock)
	}
}

func TestIfTheFirstReleaseFailsTheOrderIsReopened(t *testing.T) {
	svc, cat, _, repo := setup(CartLine{1, 3})
	o, _, _ := svc.Checkout(ctx, 7, "tok", "")
	cat.releaseErr[1] = apperr.Unavailable("product service is unreachable")

	if _, err := svc.Cancel(ctx, 7, o.ID); !errors.Is(err, apperr.ErrUnavailable) {
		t.Fatalf("err = %v, want unavailable", err)
	}
	if repo.orders[o.ID].Status != StatusPending {
		t.Error("the order must be pending again so the caller can retry")
	}
	if cat.stock[1] != 7 {
		t.Errorf("stock = %d, want 7 (still held by the order)", cat.stock[1])
	}
}

// Product 1 is returned, then product 2 fails: product 1 must be taken again so the reopened
// order and the catalogue agree.
func TestIfALaterReleaseFailsEarlierOnesAreReservedAgain(t *testing.T) {
	svc, cat, _, repo := setup(CartLine{1, 3}, CartLine{2, 2})
	o, _, _ := svc.Checkout(ctx, 7, "tok", "")
	cat.releaseErr[2] = apperr.Unavailable("product service is unreachable")

	if _, err := svc.Cancel(ctx, 7, o.ID); err == nil {
		t.Fatal("want error")
	}
	if repo.orders[o.ID].Status != StatusPending {
		t.Error("order must be reopened")
	}
	if cat.stock[1] != 7 || cat.stock[2] != 3 {
		t.Errorf("stock = %v, want 1:7 2:3 (both still held by the order)", cat.stock)
	}
}

// The loser of a same-key race can reach the cart after the winner has emptied it. That is
// a replay of the winner's order, not an "empty cart" error.
func TestALateSameKeyRequestSeesTheWinnersOrderNotAnEmptyCart(t *testing.T) {
	svc, _, carts, repo := setup()
	key := "key-late"
	winner := Order{ID: 77, UserID: 7, Status: StatusPending, Total: 1, IdempotencyKey: &key}
	carts.onGet = func() { repo.orders[77] = &winner } // the winner commits while we read the cart

	o, created, err := svc.Checkout(ctx, 7, "tok", key)

	if err != nil || created || o.ID != 77 {
		t.Fatalf("Checkout = %+v created=%v err=%v, want the winner's order", o, created, err)
	}
}
