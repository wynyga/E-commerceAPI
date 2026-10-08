// Drives one scenario against the in-browser demo shop and against the real Go services, and
// reports every place where the two answer differently (status, error message, field errors,
// response shape). It keeps the demo honest: if the services change, this shows what to update.
//
//   docker compose up -d                       (from the repo root, with AUTH_RATE_PER_MINUTE=1000)
//   ADMIN_PASSWORD=... node scripts/conformance.ts
//
// Optional: SHOP_AUTH_URL, SHOP_PRODUCT_URL, SHOP_CART_URL, SHOP_ORDER_URL, ADMIN_EMAIL.

import { DemoShop } from "../lib/demo/shop.ts";

type Reply = { status: number; json: unknown };
type Opts = { as?: string; body?: unknown; query?: Record<string, string | number>; key?: string };
type Client = {
  name: string;
  /** Signs in and returns a handle to pass as `as`. */
  login(email: string, password: string): Promise<string>;
  call(method: string, path: string, o?: Opts): Promise<Reply>;
};

// ---- the two clients ------------------------------------------------------------------------

function demoClient(): Client {
  const shop = new DemoShop();
  return {
    name: "demo",
    async login(email, password) {
      const res = shop.handle({ method: "POST", path: "/auth/login", body: { email, password } });
      if (res.session == null) throw new Error(`demo login failed for ${email}`);
      return String(res.session);
    },
    async call(method, path, o = {}) {
      const query: Record<string, string> = {};
      for (const [k, v] of Object.entries(o.query ?? {})) query[k] = String(v);
      const res = shop.handle({
        method,
        path,
        query,
        body: o.body,
        headers: o.key ? { "idempotency-key": o.key } : undefined,
        userId: o.as ? Number(o.as) : null,
      });
      return { status: res.status, json: res.json === null ? null : JSON.parse(JSON.stringify(res.json)) };
    },
  };
}

function realClient(): Client {
  const env = (k: string, d: string) => (process.env[k] ?? d).replace(/\/+$/, "");
  const base = {
    auth: env("SHOP_AUTH_URL", "http://localhost:8091"),
    cart: env("SHOP_CART_URL", "http://localhost:8092"),
    product: env("SHOP_PRODUCT_URL", "http://localhost:8093"),
    order: env("SHOP_ORDER_URL", "http://localhost:8094"),
  };
  const target = (path: string): string => {
    const [head, ...rest] = path.split("/").filter(Boolean);
    const tail = rest.length ? "/" + rest.join("/") : "";
    switch (head) {
      case "products":
        return `${base.product}/api/v1/products${tail}`;
      case "cart":
        return `${base.cart}/api/v1/cart${tail}`;
      case "orders":
        return `${base.order}/api/v1/orders${tail}`;
      case "profile":
        return `${base.auth}/api/v1/profile`;
      case "auth":
        return `${base.auth}/api/v1/${rest[0]}`;
      default:
        throw new Error(`no route for ${path}`);
    }
  };
  const tokens = new Map<string, string>();
  const client: Client = {
    name: "real",
    async login(email, password) {
      const res = await client.call("POST", "/auth/login", { body: { email, password } });
      if (res.status !== 200) throw new Error(`real login failed for ${email}: ${res.status}`);
      const token = (res.json as { token: string }).token;
      tokens.set(token, token);
      return token;
    },
    async call(method, path, o = {}) {
      const url = new URL(target(path));
      for (const [k, v] of Object.entries(o.query ?? {})) url.searchParams.set(k, String(v));
      const headers: Record<string, string> = {};
      if (o.body !== undefined) headers["Content-Type"] = "application/json";
      if (o.as) headers.Authorization = `Bearer ${o.as}`;
      if (o.key) headers["Idempotency-Key"] = o.key;
      const res = await fetch(url, { method, headers, body: o.body === undefined ? undefined : JSON.stringify(o.body) });
      const text = await res.text();
      return { status: res.status, json: text ? JSON.parse(text) : null };
    },
  };
  return client;
}

// ---- the scenario ------------------------------------------------------------------------------

type Step = { label: string; reply: Reply };

/** Runs the scenario and returns each step's reply. `tag` makes names and emails unique per run. */
async function scenario(c: Client, tag: string, adminEmail: string, adminPassword: string): Promise<Step[]> {
  const steps: Step[] = [];
  const rec = async (label: string, p: Promise<Reply>): Promise<Reply> => {
    const reply = await p;
    steps.push({ label, reply });
    return reply;
  };
  const idOf = (r: Reply) => (r.json as { id: number }).id;

  const admin = await c.login(adminEmail, adminPassword);
  const emailA = `a-${tag}@shop.test`;
  const emailB = `b-${tag}@shop.test`;

  await rec("register A", c.call("POST", "/auth/register", { body: { email: emailA, password: "long-enough-1" } }));
  await rec("register B", c.call("POST", "/auth/register", { body: { email: emailB, password: "long-enough-1" } }));
  await rec("register duplicate", c.call("POST", "/auth/register", { body: { email: emailA.toUpperCase(), password: "long-enough-1" } }));
  await rec("register invalid", c.call("POST", "/auth/register", { body: { email: "nope", password: "short" } }));
  await rec("register with role field", c.call("POST", "/auth/register", { body: { email: `c-${tag}@shop.test`, password: "long-enough-1", role: "admin" } }));
  await rec("login wrong password", c.call("POST", "/auth/login", { body: { email: emailA, password: "wrong-wrong" } }));
  await rec("login unknown email", c.call("POST", "/auth/login", { body: { email: `ghost-${tag}@shop.test`, password: "wrong-wrong" } }));
  const a = await c.login(emailA, "long-enough-1");
  const b = await c.login(emailB, "long-enough-1");
  await rec("profile", c.call("GET", "/profile", { as: a }));
  await rec("profile anonymous", c.call("GET", "/profile"));

  const mk = (name: string, price: number, stock: number) => ({ name: `${name} ${tag}`, description: "Conformance item", price, stock });
  await rec("customer creates product", c.call("POST", "/products", { as: a, body: mk("Nope", 1000, 1) }));
  await rec("anonymous creates product", c.call("POST", "/products", { body: mk("Nope", 1000, 1) }));
  await rec("invalid product", c.call("POST", "/products", { as: admin, body: { name: " ", description: "", price: 0, stock: -1 } }));
  await rec("fractional price", c.call("POST", "/products", { as: admin, body: { ...mk("Frac", 1000, 1), price: 10.5 } }));
  const p1 = await rec("create P1", c.call("POST", "/products", { as: admin, body: mk("Alpha", 10000, 5) }));
  const p2 = await rec("create P2", c.call("POST", "/products", { as: admin, body: mk("Bravo", 25000, 3) }));
  const p3 = await rec("create P3 (no stock)", c.call("POST", "/products", { as: admin, body: mk("Charlie", 5000, 0) }));
  const [id1, id2, id3] = [idOf(p1), idOf(p2), idOf(p3)];

  await rec("get product", c.call("GET", `/products/${id1}`));
  await rec("get unknown product", c.call("GET", "/products/99999999"));
  await rec("search", c.call("GET", "/products", { query: { q: `Alpha ${tag}` } }));
  await rec("sort -price page", c.call("GET", "/products", { query: { sort: "-price", limit: 2, page: 1 } }));
  await rec("update P1", c.call("PUT", `/products/${id1}`, { as: admin, body: mk("Alpha", 10000, 5) }));

  await rec("cart empty", c.call("GET", "/cart", { as: a }));
  await rec("cart anonymous", c.call("GET", "/cart"));
  await rec("add P1 x2", c.call("POST", "/cart/items", { as: a, body: { product_id: id1, quantity: 2 } }));
  await rec("add P1 x4 (over stock)", c.call("POST", "/cart/items", { as: a, body: { product_id: id1, quantity: 4 } }));
  await rec("add P3 (sold out)", c.call("POST", "/cart/items", { as: a, body: { product_id: id3, quantity: 1 } }));
  await rec("add quantity 0", c.call("POST", "/cart/items", { as: a, body: { product_id: id1, quantity: 0 } }));
  await rec("add quantity 100", c.call("POST", "/cart/items", { as: a, body: { product_id: id1, quantity: 100 } }));
  await rec("add unknown product", c.call("POST", "/cart/items", { as: a, body: { product_id: 99999999, quantity: 1 } }));
  const cartA = await rec("add P1 x1 (merge)", c.call("POST", "/cart/items", { as: a, body: { product_id: id1, quantity: 1 } }));
  const itemId = (cartA.json as { items: { id: number }[] }).items[0].id;
  await rec("patch quantity", c.call("PATCH", `/cart/items/${itemId}`, { as: a, body: { quantity: 2 } }));
  await rec("patch other user's line", c.call("PATCH", `/cart/items/${itemId}`, { as: b, body: { quantity: 1 } }));
  await rec("delete other user's line", c.call("DELETE", `/cart/items/${itemId}`, { as: b }));

  await rec("checkout empty cart", c.call("POST", "/orders", { as: b }));
  const o1 = await rec("checkout", c.call("POST", "/orders", { as: a, key: `k-${tag}` }));
  const orderId = idOf(o1);
  const replay = await rec("checkout replay", c.call("POST", "/orders", { as: a, key: `k-${tag}` }));
  steps.push({ label: "replay is the same order", reply: { status: idOf(replay) === orderId ? 200 : 500, json: null } });
  await rec("stock after checkout", c.call("GET", `/products/${id1}`));
  await rec("cart after checkout", c.call("GET", "/cart", { as: a }));
  await rec("checkout key too long", c.call("POST", "/orders", { as: a, key: "x".repeat(101) }));

  // Rollback: P1 is reserved first, then P2 turns out short, so P1 must be put back.
  await rec("refill cart P1", c.call("POST", "/cart/items", { as: a, body: { product_id: id1, quantity: 1 } }));
  await rec("refill cart P2", c.call("POST", "/cart/items", { as: a, body: { product_id: id2, quantity: 2 } }));
  await rec("admin drains P2", c.call("PUT", `/products/${id2}`, { as: admin, body: mk("Bravo", 25000, 1) }));
  await rec("checkout fails on the later line", c.call("POST", "/orders", { as: a }));
  await rec("P1 stock unchanged by the failure", c.call("GET", `/products/${id1}`));
  await rec("cart kept after the failure", c.call("GET", "/cart", { as: a }));

  await rec("get order", c.call("GET", `/orders/${orderId}`, { as: a }));
  await rec("other user reads order", c.call("GET", `/orders/${orderId}`, { as: b }));
  await rec("other user cancels order", c.call("POST", `/orders/${orderId}/cancel`, { as: b }));
  await rec("list orders", c.call("GET", "/orders", { as: a }));
  await rec("cancel order", c.call("POST", `/orders/${orderId}/cancel`, { as: a }));
  await rec("cancel again", c.call("POST", `/orders/${orderId}/cancel`, { as: a }));
  await rec("stock after cancel", c.call("GET", `/products/${id1}`));
  await rec("clear cart", c.call("DELETE", "/cart", { as: a }));
  await rec("delete product", c.call("DELETE", `/products/${id3}`, { as: admin }));
  await rec("deleted product is gone", c.call("GET", `/products/${id3}`));
  return steps;
}

// ---- comparison ----------------------------------------------------------------------------------

// `image` exists only in the demo catalogue; the Go product service has no such field.
const IGNORED = new Set(["created_at", "updated_at", "image"]);

/** A comparable description of a JSON value: keys and types, not data, with timestamps left out. */
function shape(v: unknown, depth = 0): unknown {
  if (v === null) return "null";
  if (Array.isArray(v)) return v.length === 0 ? "[]" : [shape(v[0], depth + 1)];
  if (typeof v === "object") {
    const out: Record<string, unknown> = {};
    for (const k of Object.keys(v as object).sort()) if (!IGNORED.has(k)) out[k] = shape((v as Record<string, unknown>)[k], depth + 1);
    return out;
  }
  return typeof v;
}

/** Error text with numbers masked (ids differ between the two runs). */
const message = (r: Reply) => ((r.json as { error?: string } | null)?.error ?? "").replace(/\d+/g, "N");
const fieldKeys = (r: Reply) => JSON.stringify(Object.entries((r.json as { fields?: object } | null)?.fields ?? {}).sort());

async function main() {
  const adminPassword = process.env.ADMIN_PASSWORD;
  if (!adminPassword) {
    console.error("Set ADMIN_PASSWORD to the real stack's admin password.");
    process.exit(2);
  }
  const adminEmail = process.env.ADMIN_EMAIL ?? "admin@shop.test";
  const tag = Date.now().toString(36);

  const real = await scenario(realClient(), tag, adminEmail, adminPassword);
  // The demo seeds its own admin; the scenario only needs an account with the admin role.
  const demo = await scenario(demoClient(), tag, "admin@shop.test", "demo-admin-1234");

  let diffs = 0;
  real.forEach((r, i) => {
    const d = demo[i];
    const problems: string[] = [];
    if (r.reply.status !== d.reply.status) problems.push(`status real=${r.reply.status} demo=${d.reply.status}`);
    if (message(r.reply) !== message(d.reply)) problems.push(`message real=${JSON.stringify(message(r.reply))} demo=${JSON.stringify(message(d.reply))}`);
    if (fieldKeys(r.reply) !== fieldKeys(d.reply)) problems.push(`fields real=[${fieldKeys(r.reply)}] demo=[${fieldKeys(d.reply)}]`);
    const [sr, sd] = [JSON.stringify(shape(r.reply.json)), JSON.stringify(shape(d.reply.json))];
    if (r.reply.status < 400 && sr !== sd) problems.push(`shape real=${sr} demo=${sd}`);
    if (problems.length) {
      diffs++;
      console.log(`✗ ${r.label}\n    ${problems.join("\n    ")}`);
    }
  });
  console.log(`\n${real.length} steps compared, ${diffs} differ.`);
  process.exit(diffs ? 1 : 0);
}

main().catch((e) => {
  console.error(e);
  process.exit(2);
});
