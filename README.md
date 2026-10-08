# E-commerce API

A small online-shop backend as four Go services, each with its own MySQL database:
accounts, catalogue, shopping cart, and orders. The interesting part is **checkout**: an order
touches three services and there is no shared transaction, so the order service sequences its
steps so that every failure can be undone.

```
client ─┬─► auth      register · login · JWT (roles: customer, admin)
        ├─► product   catalogue · admin-only writes · stock
        ├─► cart      one cart per user · validates against the catalogue
        └─► order     checkout · cancel · idempotent retries
                       │
                       ├─ reads the buyer's cart        (cart service, with the buyer's token)
                       └─ reserves / releases stock     (product service, internal key)
```

## Storefront

[`web/`](web) is a Next.js shop on top of these services: catalogue with search and sorting, cart,
checkout, orders and cancel, and an admin page for products. Its **demo mode** runs the whole shop in the
visitor's browser (same rules as the services, checked by a conformance script), so it deploys to Vercel
with no backend. With the four `SHOP_*_URL` variables set it talks to the real services instead, keeping the
JWT in an `httpOnly` cookie. See [`web/README.md`](web/README.md).

## How checkout stays consistent

1. **Stock is taken with one conditional `UPDATE`** (`... WHERE stock >= ?`), so two buyers can never
   both get the last unit, and stock can never go negative (a `CHECK` constraint backs it up).
2. **Reserve first, record second.** If a later line is out of stock, or saving the order fails, every
   unit already taken is released. A failed checkout leaves no stock locked and no half-order.
3. **Idempotent retries.** Send `Idempotency-Key: <anything unique>` and a repeated checkout (a timeout and
   retry, a double click) returns the original order instead of creating a second one. The database
   enforces it with a unique `(user_id, idempotency_key)`; if two identical requests race, the loser
   gives its stock back and returns the winner's order.
4. **Cancel returns stock exactly once.** The order is flipped to `cancelled` with a conditional update
   first, so only one caller ever releases the stock; if releasing fails part-way, the order is reopened
   and what was already returned is reserved again, so the caller can simply retry.
5. **Orders keep the name and price they were sold at**, so a later price change never rewrites history.
   Prices are whole rupiah stored as integers, so totals have no rounding.

## Other decisions

- **Roles.** Anyone can read the catalogue; only an `admin` can change it. Roles live in the signed token,
  and admins are created at startup from `ADMIN_EMAIL` / `ADMIN_PASSWORD`, never through registration.
- **No secrets in the repo.** Everything comes from `.env`; JWT and internal keys must be 32+ characters or
  the service refuses to start. Services connect as a dedicated database user that can only see the four
  databases, not as root. The database port is not published.
- **Errors are safe.** Only intended messages reach the client (`{"error": "..."}`, plus `fields` for
  validation); driver errors are logged and answered with a generic 500.
- **Other users' data looks missing.** Cart items and orders are always looked up with the caller's id, so
  someone else's id answers 404, exactly like one that does not exist.
- **Brute-force friction.** Login and register are rate limited per client address, and an unknown email
  costs the same time as a wrong password.
- Shared code (errors, JSON/validation helpers, JWT, config, DB + migrations, service-to-service client)
  lives in [`shared/`](shared); each service embeds its own SQL migrations and applies them on startup.

## Run it

Needs Docker.

```bash
cp .env.example .env              # then fill in real values (openssl rand -hex 32)
docker compose up -d --build
```

| Service | URL |
| --- | --- |
| auth | http://localhost:8091 |
| cart | http://localhost:8092 |
| product | http://localhost:8093 |
| order | http://localhost:8094 |

## API

All under `/api/v1`. Errors are `{ "error": "...", "fields": { ... } }`; lists are `{ items, page, limit, total, total_pages }`.

| Service | Endpoints |
| --- | --- |
| auth | `POST /register` · `POST /login` · `GET /profile` |
| product | `GET /products?q=&sort=name\|price\|-price\|newest&page=&limit=` · `GET /products/{id}` · admin: `POST /products`, `PUT /products/{id}`, `DELETE /products/{id}` |
| cart | `GET /cart` · `DELETE /cart` · `POST /cart/items` · `PATCH /cart/items/{id}` · `DELETE /cart/items/{id}` |
| order | `POST /orders` (header `Idempotency-Key`) · `GET /orders` · `GET /orders/{id}` · `POST /orders/{id}/cancel` |

A quick walk-through:

```bash
TOKEN=$(curl -s localhost:8091/api/v1/login -H 'content-type: application/json' \
  -d '{"email":"admin@shop.test","password":"<ADMIN_PASSWORD>"}' | jq -r .token)
curl -s localhost:8093/api/v1/products/ -H "authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"name":"Mug","description":"Ceramic","price":45000,"stock":10}'
```

## Tests

```bash
go test ./shared/... ./services/auth/... ./services/product/... ./services/cart/... ./services/order/...
ADMIN_PASSWORD=<your ADMIN_PASSWORD> go run ./tools/e2e     # needs the stack running; AUTH_RATE_PER_MINUTE=1000 for the full run
```

- **Unit tests** cover the checkout/cancel orchestration with in-memory fakes (rollback on short stock,
  rollback on a failed save, replays, the same-key race, partial cancel failures), the cart rules, and the
  shared pieces (validation, token verification including `alg: none`, role checks, error mapping).
- **`tools/e2e`** drives the running stack over HTTP: roles across services, validation, ownership checks,
  stock rollback, **12 parallel buyers competing for 5 units** (exactly 5 orders, stock ends at 0), and
  **10 parallel retries with one idempotency key** (one order, stock taken once). `E2E_MODE=rate` checks the
  login rate limit at its default.

## What it is not

There is no payment step or order fulfilment (an order is `pending` or `cancelled`), no API gateway (the
services are called directly; the storefront's `/api` routes only forward requests and hold the cookie), and tokens last 24 hours with no refresh. Prices are IDR only.
