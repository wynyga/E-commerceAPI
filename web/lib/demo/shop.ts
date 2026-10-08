// The whole shop, in memory. In demo mode this stands in for the four Go services and applies
// the same rules: roles, validation, the soft stock check in the cart, atomic stock reservation
// with rollback at checkout, idempotent retries, and cancel that gives stock back exactly once.
//
// It is plain TypeScript with no browser APIs, so it can be tested in Node; handlers.ts adds
// the localStorage persistence on top.

import { DEMO_ADMIN, DEMO_CUSTOMER } from "./accounts.ts";
import type { Cart, CartItem, Order, OrderItem, Paged, Product, ProductInput, Role, User } from "../types.ts";

export const MAX_PER_ITEM = 99;

type StoredUser = User & { passwordHash: string };

type StoredOrder = Order & { key?: string };

export type ShopState = {
  v: 1;
  next: { product: number; user: number; cartItem: number; order: number };
  products: Product[];
  users: StoredUser[];
  /** One cart per user, keyed by user id. */
  carts: Record<string, CartItem[]>;
  orders: StoredOrder[];
};

export type Req = {
  method: string;
  /** Path without the /api prefix, e.g. "/products/3". */
  path: string;
  query?: Record<string, string | undefined>;
  body?: unknown;
  headers?: Record<string, string | undefined>;
  /** The signed-in user's id, from the demo session. */
  userId?: number | null;
};

export type Res = {
  status: number;
  json: unknown;
  /** Set when the request changed who is signed in (null = signed out). */
  session?: number | null;
};

type Fields = Record<string, string>;

class HttpError extends Error {
  status: number;
  fields?: Fields;

  constructor(status: number, message: string, fields?: Fields) {
    super(message);
    this.status = status;
    this.fields = fields;
  }
}

const invalid = (m: string, fields?: Fields) => new HttpError(400, m, fields);
const unauthorized = (m = "missing bearer token") => new HttpError(401, m);
const forbidden = (role: Role) => new HttpError(403, `this action needs the ${role} role`);
const notFound = (what: string) => new HttpError(404, `${what} not found`);
const conflict = (m: string) => new HttpError(409, m);

// ---- Seed data ------------------------------------------------------------------------

export { DEMO_ADMIN, DEMO_CUSTOMER };

/** A tiny non-cryptographic hash. The demo has no secrets to protect; it only avoids keeping passwords as readable text. */
function weakHash(s: string): string {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return (h >>> 0).toString(16);
}

const SEED: Array<[string, string, number, number]> = [
  ["Ceramic Pour-over Dripper", "Hand-glazed cone dripper that fits a standard 02 paper filter. Dishwasher safe.", 185000, 24],
  ["Toraja Single-origin Beans 250g", "Medium roast, notes of cacao and dark fruit. Roasted weekly, whole bean.", 98000, 60],
  ["Stoneware Mug 350 ml", "Thick-walled mug that keeps a flat white warm. Speckled matte glaze.", 75000, 40],
  ["Gooseneck Kettle 800 ml", "Stainless steel kettle with a fine spout for slow, even pouring.", 420000, 12],
  ["Linen Tote Bag", "Heavy washed linen with an inside pocket. Holds two laptops and a lunch.", 135000, 30],
  ["Dotted Notebook A5", "160 pages of 100 gsm paper that takes fountain pen ink without bleeding.", 68000, 80],
  ["Brass Mechanical Pencil 0.5", "Weighted brass body with a retractable tip. Ships with spare leads.", 155000, 18],
  ["Walnut Desk Shelf", "Raised shelf that frees up desk space for a monitor and a plant.", 520000, 7],
  ["LED Desk Lamp", "Dimmable, warm-to-cool light, USB-C powered, folds flat for travel.", 340000, 15],
  ["Wireless Ergonomic Mouse", "Vertical grip that rests the wrist. Rechargeable, 60 days per charge.", 295000, 22],
  ["Cork Desk Mat", "Natural cork, 80 x 40 cm, with stitched edges. Wipes clean.", 210000, 26],
  ["Glass Water Bottle 600 ml", "Borosilicate glass with a silicone sleeve and a bamboo lid.", 120000, 35],
  ["Handmade Soy Candle", "Cedar and bergamot, 45 hours of burn time, in a reusable jar.", 89000, 3],
  ["Limited Print: Merapi Dawn", "A3 giclee print, signed and numbered. Only a handful were made.", 650000, 2],
  ["Enamel Camp Mug", "Speckled enamel, 300 ml, light enough for a daypack.", 59000, 0],
  ["Cotton Apron", "Washed canvas with adjustable neck strap and two deep pockets.", 175000, 14],
];

export function seedState(now = new Date()): ShopState {
  const at = (n: number) => new Date(now.getTime() - n * 3_600_000).toISOString();
  const products: Product[] = SEED.map(([name, description, price, stock], i) => ({
    id: i + 1,
    name,
    description,
    price,
    stock,
    created_at: at(SEED.length - i + 48),
    updated_at: at(SEED.length - i + 48),
  }));
  const users: StoredUser[] = [
    { id: 1, email: DEMO_ADMIN.email, role: "admin", created_at: at(500), passwordHash: weakHash(DEMO_ADMIN.password) },
    { id: 2, email: DEMO_CUSTOMER.email, role: "customer", created_at: at(400), passwordHash: weakHash(DEMO_CUSTOMER.password) },
  ];
  return {
    v: 1,
    next: { product: products.length + 1, user: users.length + 1, cartItem: 1, order: 1 },
    products,
    users,
    carts: {},
    orders: [],
  };
}

// ---- Validation, written to match the Go validator's messages ----------------------------

function asRecord(body: unknown): Record<string, unknown> {
  if (body === undefined || body === null) throw invalid("request body is empty");
  if (typeof body !== "object" || Array.isArray(body)) throw invalid("invalid request body");
  return body as Record<string, unknown>;
}

/** Rejects unknown fields like the real services do. */
function only(body: Record<string, unknown>, allowed: string[]) {
  for (const k of Object.keys(body)) if (!allowed.includes(k)) throw invalid("invalid request body");
}

function str(body: Record<string, unknown>, key: string): string {
  const v = body[key];
  if (v === undefined || v === null) return "";
  if (typeof v !== "string") throw invalid("invalid request body");
  return v.trim();
}

function num(body: Record<string, unknown>, key: string): number {
  const v = body[key];
  if (v === undefined || v === null) return 0;
  if (typeof v !== "number" || !Number.isInteger(v)) throw invalid("invalid request body");
  return v;
}

function throwIfAny(fields: Fields) {
  if (Object.keys(fields).length) throw invalid("validation failed", fields);
}

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

function checkCredentials(body: unknown, strict: boolean): { email: string; password: string } {
  const b = asRecord(body);
  only(b, ["email", "password"]);
  const email = str(b, "email");
  // Passwords are not trimmed: only the surrounding whitespace of text fields is.
  const password = typeof b.password === "string" ? b.password : "";
  const f: Fields = {};
  if (!email) f.email = "is required";
  else if (!EMAIL.test(email)) f.email = "must be a valid email address";
  else if (email.length > 255) f.email = "must be at most 255 characters";
  if (!password) f.password = "is required";
  else if (password.length > 72) f.password = "must be at most 72 characters";
  else if (strict && password.length < 8) f.password = "must be at least 8 characters";
  throwIfAny(f);
  return { email: email.toLowerCase(), password };
}

function checkProduct(body: unknown): ProductInput {
  const b = asRecord(body);
  only(b, ["name", "description", "price", "stock"]);
  const input = { name: str(b, "name"), description: str(b, "description"), price: num(b, "price"), stock: num(b, "stock") };
  const f: Fields = {};
  if (!input.name) f.name = "is required";
  else if (input.name.length > 255) f.name = "must be at most 255 characters";
  if (input.description.length > 5000) f.description = "must be at most 5000 characters";
  if (input.price <= 0) f.price = "must be greater than 0";
  else if (input.price > 100_000_000_000) f.price = "must be at most 100000000000";
  if (input.stock < 0) f.stock = "must be at least 0";
  else if (input.stock > 1_000_000) f.stock = "must be at most 1000000";
  throwIfAny(f);
  return input;
}

function checkQuantity(b: Record<string, unknown>): number {
  const q = num(b, "quantity");
  if (q <= 0) throw invalid("validation failed", { quantity: "must be greater than 0" });
  if (q > MAX_PER_ITEM) throw invalid("validation failed", { quantity: `must be at most ${MAX_PER_ITEM}` });
  return q;
}

function idOf(raw: string, what: string): number {
  if (!/^\d+$/.test(raw) || Number(raw) === 0) throw notFound(what);
  return Number(raw);
}

// ---- The shop ------------------------------------------------------------------------------

export class DemoShop {
  state: ShopState;
  private now: () => Date;

  constructor(state?: ShopState, now: () => Date = () => new Date()) {
    this.now = now;
    this.state = state ?? seedState(now());
  }

  /** Answers a request the way the real API would. Never throws. */
  handle(req: Req): Res {
    try {
      return this.route(req);
    } catch (e) {
      if (e instanceof HttpError) {
        return { status: e.status, json: e.fields ? { error: e.message, fields: e.fields } : { error: e.message } };
      }
      throw e;
    }
  }

  private route(req: Req): Res {
    const method = req.method.toUpperCase();
    const parts = req.path.split("?")[0].split("/").filter(Boolean);
    const [head, a, b] = parts;

    if (head === "auth" && parts.length === 2) {
      if (a === "login" && method === "POST") return this.login(req.body);
      if (a === "register" && method === "POST") return this.register(req.body);
      if (a === "logout" && method === "POST") return { status: 200, json: { ok: true }, session: null };
    }
    if (head === "profile" && parts.length === 1 && method === "GET") {
      return { status: 200, json: { user: this.requireUser(req) } };
    }

    if (head === "products") {
      if (parts.length === 1 && method === "GET") return this.listProducts(req.query ?? {});
      if (parts.length === 1 && method === "POST") return this.createProduct(req);
      if (parts.length === 2 && method === "GET") return { status: 200, json: this.product(a) };
      if (parts.length === 2 && method === "PUT") return this.updateProduct(req, a);
      if (parts.length === 2 && method === "DELETE") return this.deleteProduct(req, a);
    }

    if (head === "cart") {
      if (parts.length === 1 && method === "GET") return { status: 200, json: this.cart(this.requireUser(req).id) };
      if (parts.length === 1 && method === "DELETE") {
        delete this.state.carts[this.requireUser(req).id];
        return { status: 204, json: null };
      }
      if (a === "items" && parts.length === 2 && method === "POST") return this.addItem(req);
      if (a === "items" && parts.length === 3 && method === "PATCH") return this.setQuantity(req, b);
      if (a === "items" && parts.length === 3 && method === "DELETE") return this.removeItem(req, b);
    }

    if (head === "orders") {
      if (parts.length === 1 && method === "POST") return this.checkout(req);
      if (parts.length === 1 && method === "GET") return this.listOrders(req);
      if (parts.length === 2 && method === "GET") return { status: 200, json: this.ownOrder(this.requireUser(req).id, a) };
      if (parts.length === 3 && b === "cancel" && method === "POST") return this.cancel(req, a);
    }

    if (head === "demo" && a === "rival" && parts.length === 2 && method === "POST") return this.rival(req);

    return { status: 404, json: { error: "not found" } };
  }

  // -- auth --

  private requireUser(req: Req): User {
    const u = this.state.users.find((x) => x.id === req.userId);
    if (!u) throw unauthorized();
    return publicUser(u);
  }

  private requireAdmin(req: Req): User {
    const u = this.requireUser(req);
    if (u.role !== "admin") throw forbidden("admin");
    return u;
  }

  private register(body: unknown): Res {
    const { email, password } = checkCredentials(body, true);
    if (this.state.users.some((u) => u.email === email)) throw conflict("email is already registered");
    const u: StoredUser = {
      id: this.state.next.user++,
      email,
      role: "customer",
      created_at: this.now().toISOString(),
      passwordHash: weakHash(password),
    };
    this.state.users.push(u);
    return { status: 201, json: { user: publicUser(u) } };
  }

  private login(body: unknown): Res {
    const { email, password } = checkCredentials(body, false);
    const u = this.state.users.find((x) => x.email === email);
    if (!u || u.passwordHash !== weakHash(password)) throw unauthorized("invalid email or password");
    return { status: 200, json: { user: publicUser(u) }, session: u.id };
  }

  // -- catalogue --

  private product(raw: string): Product {
    const p = this.state.products.find((x) => x.id === idOf(raw, "product"));
    if (!p) throw notFound("product");
    return p;
  }

  private listProducts(q: Record<string, string | undefined>): Res {
    const term = (q.q ?? "").trim().toLowerCase();
    let items = this.state.products.filter((p) => !term || p.name.toLowerCase().includes(term) || p.description.toLowerCase().includes(term));
    const byId = (x: Product, y: Product) => x.id - y.id;
    const cmp: Record<string, (x: Product, y: Product) => number> = {
      price: (x, y) => x.price - y.price || byId(x, y),
      "-price": (x, y) => y.price - x.price || byId(x, y),
      newest: (x, y) => y.id - x.id,
    };
    items = [...items].sort(cmp[q.sort ?? ""] ?? ((x, y) => x.name.localeCompare(y.name) || byId(x, y)));
    return { status: 200, json: page(items, q.page, q.limit) };
  }

  private createProduct(req: Req): Res {
    this.requireAdmin(req);
    const input = checkProduct(req.body);
    const at = this.now().toISOString();
    const p: Product = { id: this.state.next.product++, ...input, created_at: at, updated_at: at };
    this.state.products.push(p);
    return { status: 201, json: p };
  }

  private updateProduct(req: Req, raw: string): Res {
    this.requireAdmin(req);
    const id = idOf(raw, "product");
    const input = checkProduct(req.body);
    const p = this.state.products.find((x) => x.id === id);
    if (!p) throw notFound("product");
    Object.assign(p, input, { updated_at: this.now().toISOString() });
    return { status: 200, json: p };
  }

  private deleteProduct(req: Req, raw: string): Res {
    this.requireAdmin(req);
    const id = idOf(raw, "product");
    const i = this.state.products.findIndex((x) => x.id === id);
    if (i < 0) throw notFound("product");
    this.state.products.splice(i, 1);
    return { status: 204, json: null };
  }

  // -- cart --

  private cart(userId: number): Cart {
    return { id: userId, user_id: userId, items: (this.state.carts[userId] ?? []).map((i) => ({ ...i })) };
  }

  private lines(userId: number): CartItem[] {
    return (this.state.carts[userId] ??= []);
  }

  /** The cart only checks stock softly; the real reservation happens at checkout. */
  private checkQuantity(p: Product, want: number) {
    if (want > MAX_PER_ITEM) throw conflict(`a cart can hold at most ${MAX_PER_ITEM} of one product`);
    if (want > p.stock) throw conflict(`only ${p.stock} of "${p.name}" in stock`);
  }

  private addItem(req: Req): Res {
    const user = this.requireUser(req);
    const b = asRecord(req.body);
    only(b, ["product_id", "quantity"]);
    const productId = num(b, "product_id");
    const f: Fields = {};
    if (productId <= 0) f.product_id = productId === 0 ? "is required" : "must be greater than 0";
    const q = num(b, "quantity");
    if (q <= 0) f.quantity = "must be greater than 0";
    else if (q > MAX_PER_ITEM) f.quantity = `must be at most ${MAX_PER_ITEM}`;
    throwIfAny(f);

    const p = this.state.products.find((x) => x.id === productId);
    if (!p) throw notFound("product");
    const lines = this.lines(user.id);
    const line = lines.find((l) => l.product_id === productId);
    this.checkQuantity(p, (line?.quantity ?? 0) + q);
    if (line) line.quantity += q;
    else lines.push({ id: this.state.next.cartItem++, cart_id: user.id, product_id: productId, quantity: q });
    return { status: 200, json: this.cart(user.id) };
  }

  private setQuantity(req: Req, raw: string): Res {
    const user = this.requireUser(req);
    const itemId = idOf(raw, "item");
    const b = asRecord(req.body);
    only(b, ["quantity"]);
    const q = checkQuantity(b);
    const line = this.lines(user.id).find((l) => l.id === itemId);
    if (!line) throw notFound("item");
    const p = this.state.products.find((x) => x.id === line.product_id);
    if (!p) throw notFound("product");
    this.checkQuantity(p, q);
    line.quantity = q;
    return { status: 200, json: this.cart(user.id) };
  }

  private removeItem(req: Req, raw: string): Res {
    const user = this.requireUser(req);
    const itemId = idOf(raw, "item");
    const lines = this.lines(user.id);
    const i = lines.findIndex((l) => l.id === itemId);
    if (i < 0) throw notFound("item");
    lines.splice(i, 1);
    return { status: 200, json: this.cart(user.id) };
  }

  // -- orders --

  private ownOrder(userId: number, raw: string): Order {
    const o = this.state.orders.find((x) => x.id === idOf(raw, "order") && x.user_id === userId);
    if (!o) throw notFound("order");
    return publicOrder(o);
  }

  private listOrders(req: Req): Res {
    const user = this.requireUser(req);
    const mine = this.state.orders.filter((o) => o.user_id === user.id).sort((x, y) => y.id - x.id);
    const q = req.query ?? {};
    const p = page(mine.map(publicOrder), q.page, q.limit);
    return { status: 200, json: p };
  }

  /**
   * Reserve stock line by line, then record the order. If a line is short, everything already
   * taken is given back, so a failed checkout leaves no stock locked and no half-order.
   * With an Idempotency-Key a repeat returns the original order (200) instead of a new one (201).
   */
  private checkout(req: Req): Res {
    const user = this.requireUser(req);
    const key = (req.headers?.["idempotency-key"] ?? "").trim();
    if (key.length > 100) throw invalid("Idempotency-Key must be at most 100 characters");
    if (key) {
      const seen = this.state.orders.find((o) => o.user_id === user.id && o.key === key);
      if (seen) return { status: 200, json: publicOrder(seen) };
    }

    const lines = [...this.lines(user.id)].sort((x, y) => x.product_id - y.product_id);
    if (lines.length === 0) throw invalid("your cart is empty");

    const taken: Array<{ product: Product; qty: number }> = [];
    const items: OrderItem[] = [];
    let total = 0;
    let orderItemId = this.state.orders.reduce((n, o) => n + o.items.length, 0);
    for (const l of lines) {
      const p = this.state.products.find((x) => x.id === l.product_id);
      if (!p) {
        this.giveBack(taken);
        throw conflict(`product ${l.product_id} is no longer available`);
      }
      if (p.stock < l.quantity) {
        this.giveBack(taken);
        throw conflict(`only ${p.stock} of "${p.name}" left in stock`);
      }
      p.stock -= l.quantity;
      taken.push({ product: p, qty: l.quantity });
      items.push({ id: ++orderItemId, product_id: p.id, name: p.name, unit_price: p.price, quantity: l.quantity });
      total += p.price * l.quantity;
    }

    const order: StoredOrder = {
      id: this.state.next.order++,
      user_id: user.id,
      status: "pending",
      total,
      items,
      created_at: this.now().toISOString(),
      ...(key ? { key } : {}),
    };
    this.state.orders.push(order);
    delete this.state.carts[user.id];
    return { status: 201, json: publicOrder(order) };
  }

  private giveBack(taken: Array<{ product: Product; qty: number }>) {
    for (const t of taken) t.product.stock += t.qty;
  }

  private cancel(req: Req, raw: string): Res {
    const user = this.requireUser(req);
    const o = this.state.orders.find((x) => x.id === idOf(raw, "order") && x.user_id === user.id);
    if (!o) throw notFound("order");
    if (o.status !== "pending") throw conflict("only a pending order can be cancelled");
    o.status = "cancelled";
    for (const it of o.items) {
      // A product deleted since the order was placed has nowhere to return its stock to.
      const p = this.state.products.find((x) => x.id === it.product_id);
      if (p) p.stock += it.quantity;
    }
    return { status: 200, json: publicOrder(o) };
  }

  // -- demo only --

  /**
   * Plays a competing buyer: drains the stock of the product with the highest id in the caller's
   * cart. Because checkout reserves lines in id order, the earlier lines are taken first and then
   * handed back when the last one turns out short, which is the rollback worth seeing.
   */
  private rival(req: Req): Res {
    const user = this.requireUser(req);
    const lines = [...this.lines(user.id)].sort((x, y) => y.product_id - x.product_id);
    const target = lines.map((l) => this.state.products.find((p) => p.id === l.product_id)).find((p): p is Product => !!p && p.stock > 0);
    if (!target) throw conflict("put something with stock in your cart first");
    const left = target.stock;
    target.stock = 0;
    return { status: 200, json: { product_id: target.id, name: target.name, bought: left } };
  }
}

// ---- helpers ------------------------------------------------------------------------------

function publicUser(u: StoredUser | User): User {
  return { id: u.id, email: u.email, role: u.role, created_at: u.created_at };
}

function publicOrder(o: StoredOrder): Order {
  const { key: _key, ...rest } = o;
  void _key;
  return { ...rest, items: o.items.map((i) => ({ ...i })) };
}

function page<T>(all: T[], rawPage?: string, rawLimit?: string): Paged<T> {
  let p = parseInt(rawPage ?? "", 10);
  let limit = parseInt(rawLimit ?? "", 10);
  if (!(p >= 1)) p = 1;
  if (!(limit >= 1)) limit = 20;
  if (limit > 100) limit = 100;
  const start = (p - 1) * limit;
  return { items: all.slice(start, start + limit), page: p, limit, total: all.length, total_pages: Math.ceil(all.length / limit) };
}
