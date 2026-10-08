// Command e2e drives a running stack (docker compose up) through its public HTTP APIs and
// checks behaviour no unit test can: roles across services, stock consistency under
// parallel checkouts, idempotent retries and rollbacks. Standard library only.
//
//	ADMIN_PASSWORD=... go run ./tools/e2e                # the full scenario
//	E2E_MODE=rate go run ./tools/e2e                     # only the login rate limit (needs AUTH_RATE_PER_MINUTE=10)
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	authURL    = env("AUTH_URL", "http://localhost:8091")
	cartURL    = env("CART_URL", "http://localhost:8092")
	productURL = env("PRODUCT_URL", "http://localhost:8093")
	orderURL   = env("ORDER_URL", "http://localhost:8094")
	hc         = &http.Client{Timeout: 20 * time.Second}
	fails      int
	runID      = fmt.Sprint(time.Now().UnixNano() % 1_000_000)
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type resp struct {
	status int
	m      map[string]any
}

func call(method, url, token string, body any, headers ...string) resp {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, url, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := hc.Do(req)
	if err != nil {
		fmt.Println("request failed:", err)
		os.Exit(2)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	r := resp{status: res.StatusCode, m: map[string]any{}}
	_ = json.Unmarshal(raw, &r.m)
	return r
}

func check(name string, ok bool, detail ...any) {
	if ok {
		fmt.Println("PASS", name)
		return
	}
	fails++
	fmt.Println("FAIL", name, detail)
}

func num(m map[string]any, k string) float64 { v, _ := m[k].(float64); return v }
func str(m map[string]any, k string) string  { v, _ := m[k].(string); return v }
func sub(m map[string]any, k string) map[string]any {
	v, _ := m[k].(map[string]any)
	return v
}

// ---- helpers ---------------------------------------------------------------

const password = "Passw0rd!x"

func newUser(tag string) (email, token string) {
	email = fmt.Sprintf("%s-%s@shop.test", tag, runID)
	if r := call("POST", authURL+"/api/v1/register", "", map[string]string{"email": email, "password": password}); r.status != 201 {
		fmt.Println("register failed:", r.status, r.m)
		os.Exit(2)
	}
	r := call("POST", authURL+"/api/v1/login", "", map[string]string{"email": email, "password": password})
	return email, str(r.m, "token")
}

func adminToken() string {
	r := call("POST", authURL+"/api/v1/login", "", map[string]string{"email": env("ADMIN_EMAIL", "admin@shop.test"), "password": os.Getenv("ADMIN_PASSWORD")})
	if r.status != 200 {
		fmt.Println("admin login failed (set ADMIN_PASSWORD):", r.status, r.m)
		os.Exit(2)
	}
	return str(r.m, "token")
}

func newProduct(admin, name string, price, stock int) int {
	r := call("POST", productURL+"/api/v1/products/", admin, map[string]any{"name": name, "description": "e2e", "price": price, "stock": stock})
	if r.status != 201 {
		fmt.Println("create product failed:", r.status, r.m)
		os.Exit(2)
	}
	return int(num(r.m, "id"))
}

func stockOf(id int) int {
	return int(num(call("GET", fmt.Sprintf("%s/api/v1/products/%d", productURL, id), "", nil).m, "stock"))
}

func addToCart(token string, productID, qty int) resp {
	return call("POST", cartURL+"/api/v1/cart/items", token, map[string]int{"product_id": productID, "quantity": qty})
}

func checkout(token string, key string) resp {
	if key == "" {
		return call("POST", orderURL+"/api/v1/orders/", token, nil)
	}
	return call("POST", orderURL+"/api/v1/orders/", token, nil, "Idempotency-Key", key)
}

// ---- scenarios -------------------------------------------------------------

func main() {
	if os.Getenv("E2E_MODE") == "rate" {
		rateLimit()
		finish()
	}
	health()
	accounts()
	admin := adminToken()
	catalogue(admin)
	cartRules(admin)
	checkoutAndCancel(admin)
	rollback(admin)
	stockRace(admin)
	idempotencyRace(admin)
	finish()
}

func finish() {
	fmt.Println("\nFAILURES:", fails)
	if fails > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}

func health() {
	for name, url := range map[string]string{"auth": authURL, "product": productURL, "cart": cartURL, "order": orderURL} {
		check("health "+name, call("GET", url+"/health", "", nil).status == 200)
	}
}

func accounts() {
	email := fmt.Sprintf("acct-%s@shop.test", runID)
	r := call("POST", authURL+"/api/v1/register", "", map[string]string{"email": email, "password": password})
	check("register 201, password never returned", r.status == 201 && sub(r.m, "user")["password"] == nil && sub(r.m, "user")["PasswordHash"] == nil, r.m)
	check("new accounts are customers", str(sub(r.m, "user"), "role") == "customer")
	r = call("POST", authURL+"/api/v1/register", "", map[string]string{"email": email, "password": password})
	check("duplicate email -> 409 without database detail", r.status == 409 && str(r.m, "error") == "email is already registered", r.m)
	r = call("POST", authURL+"/api/v1/register", "", map[string]string{"email": "nope", "password": "short"})
	f, _ := r.m["fields"].(map[string]any)
	check("invalid registration -> 400 with field errors", r.status == 400 && f["email"] != nil && f["password"] != nil, r.m)
	r = call("POST", authURL+"/api/v1/register", "", map[string]any{"email": "x@shop.test", "password": password, "role": "admin"})
	check("cannot register as admin (unknown field rejected)", r.status == 400, r.m)

	bad := call("POST", authURL+"/api/v1/login", "", map[string]string{"email": email, "password": "wrong-password"})
	ghost := call("POST", authURL+"/api/v1/login", "", map[string]string{"email": "ghost-" + runID + "@shop.test", "password": "wrong-password"})
	check("wrong password and unknown email look the same", bad.status == 401 && ghost.status == 401 && str(bad.m, "error") == str(ghost.m, "error"), bad.m, ghost.m)
	r = call("POST", authURL+"/api/v1/login", "", map[string]string{"email": "  " + email + " ", "password": password})
	check("login trims the email", r.status == 200 && str(r.m, "token") != "", r.m)
	p := call("GET", authURL+"/api/v1/profile", str(r.m, "token"), nil)
	check("profile", p.status == 200 && str(sub(p.m, "user"), "email") == email, p.m)
	check("profile needs a token", call("GET", authURL+"/api/v1/profile", "", nil).status == 401)
	check("garbage token -> 401", call("GET", authURL+"/api/v1/profile", "not.a.jwt", nil).status == 401)
}

func catalogue(admin string) {
	_, cust := newUser("cust")
	check("product list is public", call("GET", productURL+"/api/v1/products/", "", nil).status == 200)
	create := map[string]any{"name": "E2E Mug " + runID, "description": "d", "price": 45000, "stock": 10}
	check("anonymous cannot create -> 401", call("POST", productURL+"/api/v1/products/", "", create).status == 401)
	check("customer cannot create -> 403", call("POST", productURL+"/api/v1/products/", cust, create).status == 403)
	r := call("POST", productURL+"/api/v1/products/", admin, create)
	check("admin creates -> 201", r.status == 201 && num(r.m, "price") == 45000, r.m)
	id := int(num(r.m, "id"))

	for name, body := range map[string]map[string]any{
		"negative price": {"name": "x", "price": -5, "stock": 1},
		"zero price":     {"name": "x", "price": 0, "stock": 1},
		"blank name":     {"name": "   ", "price": 5, "stock": 1},
		"negative stock": {"name": "x", "price": 5, "stock": -1},
		"unknown field":  {"name": "x", "price": 5, "stock": 1, "owner": "me"},
	} {
		check("validation: "+name+" -> 400", call("POST", productURL+"/api/v1/products/", admin, body).status == 400)
	}
	check("stock 0 is allowed", call("POST", productURL+"/api/v1/products/", admin, map[string]any{"name": "Sold out " + runID, "price": 5, "stock": 0}).status == 201)

	check("customer cannot update -> 403", call("PUT", fmt.Sprintf("%s/api/v1/products/%d", productURL, id), cust, create).status == 403)
	create["price"] = 50000
	r = call("PUT", fmt.Sprintf("%s/api/v1/products/%d", productURL, id), admin, create)
	check("admin updates", r.status == 200 && num(r.m, "price") == 50000, r.m)
	check("unknown product -> 404 (not 500)", call("GET", productURL+"/api/v1/products/99999999", "", nil).status == 404)
	check("non-numeric id -> 404", call("GET", productURL+"/api/v1/products/abc", "", nil).status == 404)

	l := call("GET", productURL+"/api/v1/products/?q="+"Mug%20"+runID, "", nil)
	items, _ := l.m["items"].([]any)
	check("search finds it, with paging block", l.status == 200 && len(items) == 1 && num(l.m, "total") == 1, l.m)
	check("LIKE wildcard is literal", num(call("GET", productURL+"/api/v1/products/?q=%25", "", nil).m, "total") == 0)

	tmp := newProduct(admin, "Temp "+runID, 100, 1)
	check("customer cannot delete -> 403", call("DELETE", fmt.Sprintf("%s/api/v1/products/%d", productURL, tmp), cust, nil).status == 403)
	check("admin deletes -> 204", call("DELETE", fmt.Sprintf("%s/api/v1/products/%d", productURL, tmp), admin, nil).status == 204)
	check("deleted -> 404", call("GET", fmt.Sprintf("%s/api/v1/products/%d", productURL, tmp), "", nil).status == 404)
	check("deleting twice -> 404", call("DELETE", fmt.Sprintf("%s/api/v1/products/%d", productURL, tmp), admin, nil).status == 404)

	// the stock endpoints are for the order service only
	check("internal reserve without key -> 401", call("POST", fmt.Sprintf("%s/internal/products/%d/reserve", productURL, id), cust, map[string]int{"quantity": 1}).status == 401)
	check("internal reserve with a wrong key -> 401", call("POST", fmt.Sprintf("%s/internal/products/%d/reserve", productURL, id), "", map[string]int{"quantity": 1}, "X-Internal-Key", "guess").status == 401)
	check("stock untouched by those attempts", stockOf(id) == 10)
}

func cartRules(admin string) {
	_, alice := newUser("alice")
	_, bob := newUser("bob")
	mug := newProduct(admin, "Cart Mug "+runID, 20000, 5)

	check("cart needs login", call("GET", cartURL+"/api/v1/cart", "", nil).status == 401)
	r := addToCart(alice, mug, 2)
	items, _ := r.m["items"].([]any)
	check("add item", r.status == 200 && len(items) == 1, r.m)
	r = addToCart(alice, mug, 1)
	items, _ = r.m["items"].([]any)
	line, _ := items[0].(map[string]any)
	check("adding again accumulates on one line", len(items) == 1 && num(line, "quantity") == 3, r.m)
	itemID := int(num(line, "id"))

	check("unknown product -> 404", addToCart(alice, 99999999, 1).status == 404)
	check("more than the stock -> 409", addToCart(alice, mug, 3).status == 409) // 3 + 3 > 5
	check("quantity 0 -> 400", addToCart(alice, mug, 0).status == 400)
	check("quantity 100 -> 400", addToCart(alice, mug, 100).status == 400)

	r = call("PATCH", fmt.Sprintf("%s/api/v1/cart/items/%d", cartURL, itemID), alice, map[string]int{"quantity": 5})
	check("set quantity within stock", r.status == 200, r.m)
	check("set quantity beyond stock -> 409", call("PATCH", fmt.Sprintf("%s/api/v1/cart/items/%d", cartURL, itemID), alice, map[string]int{"quantity": 6}).status == 409)

	check("another user cannot delete the item -> 404", call("DELETE", fmt.Sprintf("%s/api/v1/cart/items/%d", cartURL, itemID), bob, nil).status == 404)
	check("another user cannot change the item -> 404", call("PATCH", fmt.Sprintf("%s/api/v1/cart/items/%d", cartURL, itemID), bob, map[string]int{"quantity": 1}).status == 404)
	still := call("GET", cartURL+"/api/v1/cart", alice, nil)
	items, _ = still.m["items"].([]any)
	check("the owner's cart is intact", len(items) == 1)
	bobCart := call("GET", cartURL+"/api/v1/cart", bob, nil)
	items, _ = bobCart.m["items"].([]any)
	check("carts are per user", len(items) == 0)

	check("owner deletes the item", call("DELETE", fmt.Sprintf("%s/api/v1/cart/items/%d", cartURL, itemID), alice, nil).status == 200)
}

func checkoutAndCancel(admin string) {
	_, carol := newUser("carol")
	_, dave := newUser("dave")
	a := newProduct(admin, "Order A "+runID, 10000, 10)
	b := newProduct(admin, "Order B "+runID, 25000, 4)

	check("empty cart cannot check out -> 400", checkout(carol, "").status == 400)
	addToCart(carol, a, 2)
	addToCart(carol, b, 1)
	r := checkout(carol, "")
	check("checkout -> 201 pending, total 45000", r.status == 201 && str(r.m, "status") == "pending" && num(r.m, "total") == 2*10000+25000, r.m)
	orderID := int(num(r.m, "id"))
	check("stock was taken", stockOf(a) == 8 && stockOf(b) == 3, stockOf(a), stockOf(b))
	cart := call("GET", cartURL+"/api/v1/cart", carol, nil)
	items, _ := cart.m["items"].([]any)
	check("cart was emptied", len(items) == 0)

	// the order keeps the price it was sold at
	call("PUT", fmt.Sprintf("%s/api/v1/products/%d", productURL, a), admin, map[string]any{"name": "Order A " + runID, "price": 99999, "stock": 8})
	g := call("GET", fmt.Sprintf("%s/api/v1/orders/%d", orderURL, orderID), carol, nil)
	oi, _ := g.m["items"].([]any)
	first, _ := oi[0].(map[string]any)
	check("order keeps its price snapshot after a price change", g.status == 200 && num(g.m, "total") == 45000 && num(first, "unit_price") == 10000, g.m)

	check("another user cannot read the order -> 404", call("GET", fmt.Sprintf("%s/api/v1/orders/%d", orderURL, orderID), dave, nil).status == 404)
	check("another user cannot cancel it -> 404", call("POST", fmt.Sprintf("%s/api/v1/orders/%d/cancel", orderURL, orderID), dave, nil).status == 404)
	check("their failed cancel changed no stock", stockOf(a) == 8 && stockOf(b) == 3)

	l := call("GET", orderURL+"/api/v1/orders/", carol, nil)
	li, _ := l.m["items"].([]any)
	check("order list is the caller's own", l.status == 200 && len(li) == 1 && num(l.m, "total") == 1, l.m)

	c := call("POST", fmt.Sprintf("%s/api/v1/orders/%d/cancel", orderURL, orderID), carol, nil)
	check("cancel -> cancelled", c.status == 200 && str(c.m, "status") == "cancelled", c.m)
	check("stock returned exactly once", stockOf(a) == 10 && stockOf(b) == 4, stockOf(a), stockOf(b))
	check("cancelling twice -> 409", call("POST", fmt.Sprintf("%s/api/v1/orders/%d/cancel", orderURL, orderID), carol, nil).status == 409)
	check("a double cancel did not return stock twice", stockOf(a) == 10 && stockOf(b) == 4)
	check("unknown order -> 404", call("GET", orderURL+"/api/v1/orders/99999999", carol, nil).status == 404)
}

// Product Q (lower id) is reserved first, then product P runs short: Q's stock must come back.
func rollback(admin string) {
	_, erin := newUser("erin")
	_, frank := newUser("frank")
	q := newProduct(admin, "Rollback Q "+runID, 1000, 50)
	p := newProduct(admin, "Rollback P "+runID, 2000, 3)

	addToCart(erin, q, 5)
	addToCart(erin, p, 3) // fine while stock is 3
	addToCart(frank, p, 2)
	check("frank buys 2 of P first", checkout(frank, "").status == 201 && stockOf(p) == 1)

	r := checkout(erin, "") // erin still wants 3 of P, only 1 left
	check("short stock -> 409 naming the shortage", r.status == 409, r.m)
	check("the earlier line (Q) was given back", stockOf(q) == 50, stockOf(q))
	check("P untouched by the failed checkout", stockOf(p) == 1)
	l := call("GET", orderURL+"/api/v1/orders/", erin, nil)
	check("no order was created", num(l.m, "total") == 0, l.m)
	cart := call("GET", cartURL+"/api/v1/cart", erin, nil)
	items, _ := cart.m["items"].([]any)
	check("the cart is kept so erin can adjust it", len(items) == 2)
}

// 12 buyers, 5 units: exactly 5 orders, never a negative stock.
func stockRace(admin string) {
	const buyers, units = 12, 5
	p := newProduct(admin, "Race "+runID, 7000, units)
	tokens := make([]string, buyers)
	for i := range tokens {
		_, tokens[i] = newUser(fmt.Sprintf("racer%d", i))
		if r := addToCart(tokens[i], p, 1); r.status != 200 {
			fmt.Println("setup failed:", r.status, r.m)
			os.Exit(2)
		}
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	codes := make([]int, buyers)
	for i := range tokens {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			codes[i] = checkout(tokens[i], "").status
		}(i)
	}
	close(start)
	wg.Wait()

	created, conflicts := 0, 0
	for _, c := range codes {
		switch c {
		case 201:
			created++
		case 409:
			conflicts++
		}
	}
	check(fmt.Sprintf("%d parallel buyers, %d units: exactly %d orders", buyers, units, units), created == units && conflicts == buyers-units, codes)
	check("stock ends at exactly 0, never negative", stockOf(p) == 0, stockOf(p))
}

// One user, one cart, ten parallel retries with the same key: one order, stock taken once.
func idempotencyRace(admin string) {
	_, gina := newUser("gina")
	p := newProduct(admin, "Idem "+runID, 3000, 10)
	addToCart(gina, p, 2)

	const tries = 10
	var wg sync.WaitGroup
	start := make(chan struct{})
	rs := make([]resp, tries)
	for i := 0; i < tries; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			rs[i] = checkout(gina, "retry-"+runID)
		}(i)
	}
	close(start)
	wg.Wait()

	created, ids := 0, map[int]bool{}
	for _, r := range rs {
		if r.status == 201 {
			created++
		}
		if r.status == 201 || r.status == 200 {
			ids[int(num(r.m, "id"))] = true
		}
	}
	check("exactly one of ten parallel same-key checkouts creates the order", created == 1, created)
	check("every successful answer is the same order", len(ids) == 1, ids)
	l := call("GET", orderURL+"/api/v1/orders/", gina, nil)
	check("only one order exists for the user", num(l.m, "total") == 1, l.m)
	check("stock was taken once (10 - 2 = 8)", stockOf(p) == 8, stockOf(p))

	again := checkout(gina, "retry-"+runID)
	check("a later retry replays with 200", again.status == 200 && len(ids) == 1 && ids[int(num(again.m, "id"))], again.status, again.m)
	check("the replay took no stock", stockOf(p) == 8)
}

func rateLimit() {
	got := map[int]int{}
	for i := 0; i < 15; i++ {
		got[call("POST", authURL+"/api/v1/login", "", map[string]string{"email": "rate@shop.test", "password": "wrong-password"}).status]++
	}
	check("login is rate limited after 10 attempts/min", got[429] > 0 && got[401] >= 10, got)
}
