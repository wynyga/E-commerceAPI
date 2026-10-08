// Checks that the in-browser demo shop follows the same rules as the Go services.
// Run with:  npm test   (Node's built-in test runner, no extra dependencies)

import assert from "node:assert/strict";
import { beforeEach, describe, test } from "node:test";

import type { Cart, Order, Paged, Product, User } from "../types.ts";
import { DEMO_ADMIN, DEMO_CUSTOMER, DemoShop, type Res } from "./shop.ts";

let shop: DemoShop;
let seq = 0;

beforeEach(() => {
  seq = 0;
  // A fixed clock keeps timestamps deterministic.
  shop = new DemoShop(undefined, () => new Date(Date.UTC(2026, 0, 1, 12, 0, seq++)));
});

type Opts = { as?: number | null; body?: unknown; query?: Record<string, string>; key?: string };

function call(method: string, path: string, o: Opts = {}): Res {
  return shop.handle({
    method,
    path,
    query: o.query,
    body: o.body,
    headers: o.key ? { "idempotency-key": o.key } : undefined,
    userId: o.as ?? null,
  });
}

function login(creds: { email: string; password: string }): number {
  const res = call("POST", "/auth/login", { body: creds });
  assert.equal(res.status, 200);
  assert.ok(res.session);
  return res.session;
}

const ADMIN = () => login(DEMO_ADMIN);
const BUYER = () => login(DEMO_CUSTOMER);
const product = (id: number) => call("GET", `/products/${id}`).json as Product;
const stock = (id: number) => product(id).stock;
const errorOf = (r: Res) => (r.json as { error: string }).error;

function add(as: number, productId: number, quantity: number): Res {
  return call("POST", "/cart/items", { as, body: { product_id: productId, quantity } });
}

// Seed ids used below: 1 dripper (24), 2 beans (60), 13 candle (3), 14 print (2), 15 enamel mug (0 in stock).

describe("accounts", () => {
  test("login returns the user and starts a session; the password never leaves", () => {
    const res = call("POST", "/auth/login", { body: DEMO_CUSTOMER });
    assert.equal(res.status, 200);
    const user = (res.json as { user: User & { passwordHash?: string } }).user;
    assert.equal(user.email, DEMO_CUSTOMER.email);
    assert.equal(user.role, "customer");
    assert.equal(user.passwordHash, undefined);
  });

  test("a wrong password and an unknown email give the same 401", () => {
    const a = call("POST", "/auth/login", { body: { email: DEMO_CUSTOMER.email, password: "nope-nope" } });
    const b = call("POST", "/auth/login", { body: { email: "ghost@shop.test", password: "nope-nope" } });
    assert.equal(a.status, 401);
    assert.deepEqual(a.json, b.json);
    assert.equal(a.session, undefined);
  });

  test("register creates a customer who can then sign in; email is case-insensitive and unique", () => {
    const res = call("POST", "/auth/register", { body: { email: "New@Shop.Test", password: "long-enough-1" } });
    assert.equal(res.status, 201);
    assert.equal((res.json as { user: User }).user.role, "customer");
    assert.equal(call("POST", "/auth/login", { body: { email: "new@shop.test", password: "long-enough-1" } }).status, 200);
    assert.equal(call("POST", "/auth/register", { body: { email: "new@shop.test", password: "long-enough-1" } }).status, 409);
  });

  test("register validates the email and password length", () => {
    const res = call("POST", "/auth/register", { body: { email: "not-an-email", password: "short" } });
    assert.equal(res.status, 400);
    const fields = (res.json as { fields: Record<string, string> }).fields;
    assert.match(fields.email, /valid email/);
    assert.match(fields.password, /at least 8/);
  });

  test("a body cannot set its own role, and unknown fields are refused", () => {
    const res = call("POST", "/auth/register", { body: { email: "x@shop.test", password: "long-enough-1", role: "admin" } });
    assert.equal(res.status, 400);
  });

  test("profile needs a session", () => {
    assert.equal(call("GET", "/profile").status, 401);
    assert.equal(call("GET", "/profile", { as: BUYER() }).status, 200);
  });

  test("logout clears the session", () => {
    assert.equal(call("POST", "/auth/logout", { as: BUYER() }).session, null);
  });
});

describe("catalogue", () => {
  test("is public and sorted by name by default", () => {
    const res = call("GET", "/products");
    assert.equal(res.status, 200);
    const names = (res.json as Paged<Product>).items.map((p) => p.name);
    assert.deepEqual(names, [...names].sort((x, y) => x.localeCompare(y)));
  });

  test("pages with a total and total_pages", () => {
    const first = call("GET", "/products", { query: { limit: "5" } }).json as Paged<Product>;
    assert.equal(first.items.length, 5);
    assert.equal(first.total, 16);
    assert.equal(first.total_pages, 4);
    const last = call("GET", "/products", { query: { limit: "5", page: "4" } }).json as Paged<Product>;
    assert.equal(last.items.length, 1);
    const beyond = call("GET", "/products", { query: { limit: "5", page: "9" } }).json as Paged<Product>;
    assert.equal(beyond.items.length, 0);
  });

  test("searches name and description", () => {
    const res = call("GET", "/products", { query: { q: "linen" } }).json as Paged<Product>;
    assert.ok(res.total >= 1);
    assert.ok(res.items.every((p) => /linen/i.test(p.name + p.description)));
  });

  test("sorts by price both ways", () => {
    const asc = (call("GET", "/products", { query: { sort: "price" } }).json as Paged<Product>).items.map((p) => p.price);
    const desc = (call("GET", "/products", { query: { sort: "-price" } }).json as Paged<Product>).items.map((p) => p.price);
    assert.deepEqual(asc, [...asc].sort((x, y) => x - y));
    assert.deepEqual(desc, [...desc].sort((x, y) => y - x));
  });

  test("an unknown id is 404, and so is a non-numeric one", () => {
    assert.equal(call("GET", "/products/9999").status, 404);
    assert.equal(call("GET", "/products/abc").status, 404);
  });
});

describe("product management is admin-only", () => {
  const body = { name: "Teapot", description: "Cast iron", price: 250000, stock: 5 };

  test("anonymous is 401, a customer is 403", () => {
    assert.equal(call("POST", "/products", { body }).status, 401);
    assert.equal(call("POST", "/products", { as: BUYER(), body }).status, 403);
    assert.equal(call("PUT", "/products/1", { as: BUYER(), body }).status, 403);
    assert.equal(call("DELETE", "/products/1", { as: BUYER() }).status, 403);
  });

  test("an admin can create, update and delete", () => {
    const admin = ADMIN();
    const created = call("POST", "/products", { as: admin, body });
    assert.equal(created.status, 201);
    const id = (created.json as Product).id;
    const updated = call("PUT", `/products/${id}`, { as: admin, body: { ...body, price: 260000 } });
    assert.equal((updated.json as Product).price, 260000);
    assert.equal(call("DELETE", `/products/${id}`, { as: admin }).status, 204);
    assert.equal(call("GET", `/products/${id}`).status, 404);
  });

  test("validation reports each bad field", () => {
    const res = call("POST", "/products", { as: ADMIN(), body: { name: "  ", description: "", price: 0, stock: -1 } });
    assert.equal(res.status, 400);
    const fields = (res.json as { fields: Record<string, string> }).fields;
    assert.deepEqual(Object.keys(fields).sort(), ["name", "price", "stock"]);
  });

  test("prices must be whole rupiah", () => {
    assert.equal(call("POST", "/products", { as: ADMIN(), body: { ...body, price: 100.5 } }).status, 400);
  });
});

describe("cart", () => {
  test("needs a session", () => {
    assert.equal(call("GET", "/cart").status, 401);
  });

  test("adding the same product twice merges into one line", () => {
    const u = BUYER();
    add(u, 1, 2);
    const cart = add(u, 1, 3).json as Cart;
    assert.equal(cart.items.length, 1);
    assert.equal(cart.items[0].quantity, 5);
  });

  test("refuses more than is in stock, naming the product and the stock", () => {
    const res = add(BUYER(), 14, 3);
    assert.equal(res.status, 409);
    assert.match(errorOf(res), /only 2 of "Limited Print: Merapi Dawn" in stock/);
  });

  test("the stock check counts what the cart already holds", () => {
    const u = BUYER();
    assert.equal(add(u, 14, 2).status, 200);
    assert.equal(add(u, 14, 1).status, 409);
  });

  test("a product with no stock cannot be added", () => {
    assert.equal(add(BUYER(), 15, 1).status, 409);
  });

  test("quantity is validated", () => {
    const u = BUYER();
    assert.equal(add(u, 1, 0).status, 400);
    assert.equal(add(u, 1, 100).status, 400);
    assert.equal(add(u, 9999, 1).status, 404);
  });

  test("quantity can be changed and the line removed", () => {
    const u = BUYER();
    const item = (add(u, 2, 1).json as Cart).items[0];
    const changed = call("PATCH", `/cart/items/${item.id}`, { as: u, body: { quantity: 7 } }).json as Cart;
    assert.equal(changed.items[0].quantity, 7);
    assert.equal((call("DELETE", `/cart/items/${item.id}`, { as: u }).json as Cart).items.length, 0);
  });

  test("someone else's line looks missing", () => {
    const item = (add(BUYER(), 2, 1).json as Cart).items[0];
    const other = call("POST", "/auth/register", { body: { email: "other@shop.test", password: "long-enough-1" } });
    const otherId = (other.json as { user: User }).user.id;
    assert.equal(call("PATCH", `/cart/items/${item.id}`, { as: otherId, body: { quantity: 2 } }).status, 404);
    assert.equal(call("DELETE", `/cart/items/${item.id}`, { as: otherId }).status, 404);
  });

  test("DELETE /cart empties it", () => {
    const u = BUYER();
    add(u, 2, 1);
    assert.equal(call("DELETE", "/cart", { as: u }).status, 204);
    assert.equal((call("GET", "/cart", { as: u }).json as Cart).items.length, 0);
  });
});

describe("checkout", () => {
  test("takes stock, totals the lines, empties the cart", () => {
    const u = BUYER();
    add(u, 2, 3); // beans 98.000 x3
    add(u, 3, 1); // mug 75.000
    const before = [stock(2), stock(3)];
    const res = call("POST", "/orders", { as: u });
    assert.equal(res.status, 201);
    const o = res.json as Order;
    assert.equal(o.status, "pending");
    assert.equal(o.total, 98000 * 3 + 75000);
    assert.equal(o.items.length, 2);
    assert.deepEqual([stock(2), stock(3)], [before[0] - 3, before[1] - 1]);
    assert.equal((call("GET", "/cart", { as: u }).json as Cart).items.length, 0);
  });

  test("an empty cart is a 400", () => {
    const res = call("POST", "/orders", { as: BUYER() });
    assert.equal(res.status, 400);
    assert.equal(errorOf(res), "your cart is empty");
  });

  test("the order keeps the name and price it was sold at", () => {
    const u = BUYER();
    add(u, 1, 1);
    const o = call("POST", "/orders", { as: u }).json as Order;
    call("PUT", "/products/1", { as: ADMIN(), body: { name: "Renamed", description: "x", price: 1, stock: 10 } });
    const again = call("GET", `/orders/${o.id}`, { as: u }).json as Order;
    assert.equal(again.items[0].name, "Ceramic Pour-over Dripper");
    assert.equal(again.items[0].unit_price, 185000);
    assert.equal(again.total, 185000);
  });

  test("a short line fails the whole checkout and gives back what was already taken", () => {
    const u = BUYER();
    add(u, 1, 4); // fine
    add(u, 14, 2); // fine at this moment
    const rival = call("POST", "/demo/rival", { as: u });
    assert.equal(rival.status, 200);
    assert.equal((rival.json as { product_id: number }).product_id, 14);

    const before = [stock(1), stock(14)];
    const res = call("POST", "/orders", { as: u });
    assert.equal(res.status, 409);
    assert.match(errorOf(res), /only 0 of "Limited Print: Merapi Dawn" left in stock/);
    // product 1 was reserved first, then released again
    assert.deepEqual([stock(1), stock(14)], before);
    assert.equal((call("GET", "/orders", { as: u }).json as Paged<Order>).total, 0);
    // and the cart is untouched so the buyer can fix it
    assert.equal((call("GET", "/cart", { as: u }).json as Cart).items.length, 2);
  });

  test("the last unit goes to exactly one of two buyers", () => {
    const a = BUYER();
    const b = (call("POST", "/auth/register", { body: { email: "b@shop.test", password: "long-enough-1" } }).json as { user: User }).user.id;
    add(a, 14, 2);
    add(b, 14, 2);
    const first = call("POST", "/orders", { as: a });
    const second = call("POST", "/orders", { as: b });
    assert.equal(first.status, 201);
    assert.equal(second.status, 409);
    assert.equal(stock(14), 0);
  });

  test("a retry with the same Idempotency-Key returns the original order and takes stock once", () => {
    const u = BUYER();
    add(u, 2, 2);
    const first = call("POST", "/orders", { as: u, key: "k-1" });
    assert.equal(first.status, 201);
    const left = stock(2);
    const retry = call("POST", "/orders", { as: u, key: "k-1" });
    assert.equal(retry.status, 200);
    assert.equal((retry.json as Order).id, (first.json as Order).id);
    assert.equal(stock(2), left);
    assert.equal((call("GET", "/orders", { as: u }).json as Paged<Order>).total, 1);
  });

  test("a different key is a different order, and keys are per user", () => {
    const u = BUYER();
    add(u, 2, 1);
    const a = call("POST", "/orders", { as: u, key: "k-1" }).json as Order;
    add(u, 2, 1);
    const b = call("POST", "/orders", { as: u, key: "k-2" }).json as Order;
    assert.notEqual(a.id, b.id);

    const other = (call("POST", "/auth/register", { body: { email: "c@shop.test", password: "long-enough-1" } }).json as { user: User }).user.id;
    add(other, 2, 1);
    const c = call("POST", "/orders", { as: other, key: "k-1" });
    assert.equal(c.status, 201);
    assert.notEqual((c.json as Order).id, a.id);
  });

  test("an over-long key is refused", () => {
    const u = BUYER();
    add(u, 2, 1);
    assert.equal(call("POST", "/orders", { as: u, key: "x".repeat(101) }).status, 400);
  });
});

describe("orders", () => {
  function placed(u: number, productId = 2, qty = 2): Order {
    add(u, productId, qty);
    return call("POST", "/orders", { as: u }).json as Order;
  }

  test("are listed newest first", () => {
    const u = BUYER();
    const a = placed(u);
    const b = placed(u);
    const list = call("GET", "/orders", { as: u }).json as Paged<Order>;
    assert.deepEqual(list.items.map((o) => o.id), [b.id, a.id]);
  });

  test("cancel returns the stock exactly once", () => {
    const u = BUYER();
    const before = stock(2);
    const o = placed(u, 2, 5);
    assert.equal(stock(2), before - 5);
    const res = call("POST", `/orders/${o.id}/cancel`, { as: u });
    assert.equal(res.status, 200);
    assert.equal((res.json as Order).status, "cancelled");
    assert.equal(stock(2), before);

    const again = call("POST", `/orders/${o.id}/cancel`, { as: u });
    assert.equal(again.status, 409);
    assert.equal(errorOf(again), "only a pending order can be cancelled");
    assert.equal(stock(2), before);
  });

  test("another user's order looks missing for read and cancel", () => {
    const o = placed(BUYER());
    const other = (call("POST", "/auth/register", { body: { email: "d@shop.test", password: "long-enough-1" } }).json as { user: User }).user.id;
    assert.equal(call("GET", `/orders/${o.id}`, { as: other }).status, 404);
    assert.equal(call("POST", `/orders/${o.id}/cancel`, { as: other }).status, 404);
  });

  test("need a session", () => {
    assert.equal(call("GET", "/orders").status, 401);
    assert.equal(call("POST", "/orders").status, 401);
  });
});

describe("state", () => {
  test("survives a JSON round trip (that is how it is persisted)", () => {
    const u = BUYER();
    add(u, 2, 2);
    call("POST", "/orders", { as: u });
    const copy = new DemoShop(JSON.parse(JSON.stringify(shop.state)));
    const res = copy.handle({ method: "GET", path: "/orders", userId: u });
    assert.equal((res.json as Paged<Order>).total, 1);
  });

  test("the idempotency key is never exposed on an order", () => {
    const u = BUYER();
    add(u, 2, 1);
    const o = call("POST", "/orders", { as: u, key: "secret-ish" }).json as Order & { key?: string };
    assert.equal(o.key, undefined);
  });
});
