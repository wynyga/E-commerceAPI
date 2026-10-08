// The browser-side API client. In normal mode every call goes to this app's own /api proxy,
// which adds the session token from an httpOnly cookie and forwards to the right Go service.
// In demo mode the same calls are answered in the browser by lib/demo, so the UI code never
// knows the difference.

import { demoRequest } from "./demo/handlers";
import type { Cart, Order, Paged, Product, ProductInput, ProductQuery, User } from "./types";

export class ApiError extends Error {
  status: number;
  /** Per-field validation messages, keyed by the request field name (snake_case). */
  fields?: Record<string, string>;

  constructor(message: string, status: number, fields?: Record<string, string>) {
    super(message);
    this.status = status;
    this.fields = fields;
  }
}

let demo = false;

/** Called once by the layout, before any request is made. */
export function configureApi(options: { demo: boolean }) {
  demo = options.demo;
}

export const isDemo = () => demo;

/** Fired when a signed-in call comes back 401, so the session UI can drop the stale user. */
export const UNAUTHORIZED_EVENT = "shop:unauthorized";

type Params = Record<string, string | number | undefined>;

function qs(params?: Params): string {
  if (!params) return "";
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") sp.set(k, String(v));
  }
  const s = sp.toString();
  return s ? `?${s}` : "";
}

type Raw = { status: number; json: unknown };

async function transport(method: string, path: string, body?: unknown, headers?: Record<string, string>): Promise<Raw> {
  if (demo) return demoRequest(method, path, body, headers);
  try {
    const res = await fetch(`/api${path}`, {
      method,
      headers: { ...(body === undefined ? {} : { "Content-Type": "application/json" }), ...headers },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    const json = await res.json().catch(() => null);
    return { status: res.status, json };
  } catch {
    throw new ApiError("Cannot reach the server. Check your connection and try again.", 0);
  }
}

type Problem = { error?: string; fields?: Record<string, string> };

async function request<T>(method: string, path: string, body?: unknown, headers?: Record<string, string>): Promise<T> {
  const { status, json } = await transport(method, path, body, headers);

  if (status === 401 && typeof window !== "undefined" && path !== "/profile" && !path.startsWith("/auth/")) {
    window.dispatchEvent(new Event(UNAUTHORIZED_EVENT));
  }
  if (status >= 400) {
    const p = (json ?? {}) as Problem;
    throw new ApiError(p.error || `Request failed (${status})`, status, p.fields && Object.keys(p.fields).length ? p.fields : undefined);
  }
  return json as T;
}

export const api = {
  login: (email: string, password: string) => request<{ user: User }>("POST", "/auth/login", { email, password }),
  register: (email: string, password: string) => request<{ user: User }>("POST", "/auth/register", { email, password }),
  logout: () => request<{ ok: true }>("POST", "/auth/logout").catch(() => ({ ok: true as const })),
  me: () => request<{ user: User }>("GET", "/profile").then((r) => r.user),

  products: {
    list: (q: ProductQuery = {}) => request<Paged<Product>>("GET", `/products${qs(q as Params)}`),
    get: (id: number) => request<Product>("GET", `/products/${id}`),
    create: (p: ProductInput) => request<Product>("POST", "/products", p),
    update: (id: number, p: ProductInput) => request<Product>("PUT", `/products/${id}`, p),
    remove: (id: number) => request<null>("DELETE", `/products/${id}`),
  },

  cart: {
    get: () => request<Cart>("GET", "/cart"),
    add: (productId: number, quantity: number) => request<Cart>("POST", "/cart/items", { product_id: productId, quantity }),
    setQuantity: (itemId: number, quantity: number) => request<Cart>("PATCH", `/cart/items/${itemId}`, { quantity }),
    remove: (itemId: number) => request<Cart>("DELETE", `/cart/items/${itemId}`),
    clear: () => request<null>("DELETE", "/cart"),
  },

  orders: {
    list: (page = 1) => request<Paged<Order>>("GET", `/orders${qs({ page, limit: 10 })}`),
    get: (id: number) => request<Order>("GET", `/orders/${id}`),
    /** The key makes a retry safe: the same key always yields the same order. */
    checkout: (idempotencyKey: string) => request<Order>("POST", "/orders", undefined, { "Idempotency-Key": idempotencyKey }),
    cancel: (id: number) => request<Order>("POST", `/orders/${id}/cancel`),
  },

  /** Demo mode only: plays a competing buyer so the stock rollback at checkout can be seen. */
  rival: () => request<{ product_id: number; name: string; bought: number }>("POST", "/demo/rival"),
};
