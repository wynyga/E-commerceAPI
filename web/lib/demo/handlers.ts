// Browser side of the demo: keeps the DemoShop in localStorage so every visitor has a private,
// resettable shop, and answers the API client's requests from it.

import { DemoShop, type ShopState } from "./shop";

const STATE_KEY = "shop-demo:state:v1";
const SESSION_KEY = "shop-demo:session:v1";

// Storage can be blocked (private windows, strict settings); the demo then lives in memory for the visit.
const memory = new Map<string, string>();

function read(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return memory.get(key) ?? null;
  }
}

function write(key: string, value: string | null) {
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, value);
  } catch {
    if (value === null) memory.delete(key);
    else memory.set(key, value);
  }
}

/** Reads the shop fresh each time, so two open tabs never disagree. */
function load(): DemoShop {
  const raw = read(STATE_KEY);
  if (raw) {
    try {
      const state = JSON.parse(raw) as ShopState;
      if (state?.v === 1 && Array.isArray(state.products)) return new DemoShop(state);
    } catch {
      // fall through to a fresh shop
    }
  }
  return new DemoShop();
}

function sessionUser(): number | null {
  const n = Number(read(SESSION_KEY));
  return Number.isInteger(n) && n > 0 ? n : null;
}

export async function demoRequest(
  method: string,
  path: string,
  body?: unknown,
  headers?: Record<string, string>,
): Promise<{ status: number; json: unknown }> {
  const url = new URL(path, "http://demo.local");
  const query: Record<string, string> = {};
  url.searchParams.forEach((v, k) => (query[k] = v));

  const shop = load();
  const res = shop.handle({
    method,
    path: url.pathname,
    query,
    body,
    headers: Object.fromEntries(Object.entries(headers ?? {}).map(([k, v]) => [k.toLowerCase(), v])),
    userId: sessionUser(),
  });

  if (method.toUpperCase() !== "GET") write(STATE_KEY, JSON.stringify(shop.state));
  if (res.session !== undefined) write(SESSION_KEY, res.session === null ? null : String(res.session));

  // The change is applied before the pause, as if the request had already reached the server:
  // leaving the page mid-request can't lose it. The pause only keeps loading states visible.
  await new Promise((r) => setTimeout(r, 90 + Math.random() * 110));
  return { status: res.status, json: res.json };
}

/** Throws the visitor's data away; the next request starts from the sample shop again. */
export function resetDemo() {
  write(STATE_KEY, null);
  write(SESSION_KEY, null);
}
