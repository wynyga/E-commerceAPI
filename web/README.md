# Kedai Kecil — storefront

A Next.js storefront for the four Go services in this repo: browse and search the catalogue, keep a
cart, check out, review and cancel orders, and (as an admin) manage products.

It runs in two modes, chosen by environment variables:

| Mode | When | What happens |
| --- | --- | --- |
| **Demo** | none of the `SHOP_*_URL` variables are set | The whole shop runs in the visitor's browser, with sample data kept in `localStorage`. Nothing is sent anywhere, every visitor has a private shop, and "Reset shop" starts over. This is what the hosted demo uses. |
| **Real** | all four `SHOP_*_URL` variables are set | The browser talks only to this app's `/api` routes. They forward to the right Go service and keep the JWT in an `httpOnly` cookie, so page scripts never see it. |

The UI code is the same in both modes. Only the transport underneath differs.

## Run it

```bash
cd web
npm install
npm run dev            # http://localhost:3003, demo mode
```

To use the real services, start the stack from the repo root (`docker compose up -d`), then:

```bash
cp .env.example .env.local     # the four SHOP_*_URL values point at the compose ports
npm run dev
```

Environment variables are read when the app is built, so set them before `npm run build`.

## Demo mode

`lib/demo/shop.ts` is a small in-memory shop that follows the same rules as the Go services:

- roles (customer / admin), admin-only catalogue writes, validation messages like the real ones
- the cart's soft stock check, the 99-per-line cap, someone else's cart line looks missing
- checkout reserves stock line by line and **puts everything back if a later line is short**
- `Idempotency-Key` retries return the original order (200) instead of a second one (201)
- cancel returns stock exactly once; orders keep the name and price they were sold at

`lib/demo/handlers.ts` stores that state in `localStorage` and answers the API client's requests.

Two accounts exist from the start (one click fills the sign-in form in demo mode):
`demo@shop.test` (customer) and `admin@shop.test` (admin). The cart page also has a demo-only button,
"Another shopper buys it first", which drains the stock of the last product in your cart so you can
see a checkout fail cleanly.

## Product photos

The demo catalogue uses photos from Wikimedia Commons (CC BY, CC BY-SA, CC0 or public domain), resized and kept in
`public/products/`. Each product page credits its photographer and licence, and
[`public/products/CREDITS.md`](public/products/CREDITS.md) lists them all. The Go product service has no image field,
so `image` exists only in the demo catalogue; products from the real services, and any an admin adds in the demo,
get generated artwork (a gradient with the product's initial) instead.

## Tests

```bash
npm test                # 39 unit tests of the demo shop, Node's built-in runner
npm run typecheck
```

### Keeping the demo honest

`scripts/conformance.ts` runs one 56-step scenario against the demo shop **and** against the real
services, and prints every difference in status, error message, field errors or response shape.

```bash
# from the repo root: AUTH_RATE_PER_MINUTE=1000 docker compose up -d --build
ADMIN_PASSWORD=<the stack's ADMIN_PASSWORD> node scripts/conformance.ts
```

At the time of writing the two agree on all 56 steps. The in-browser flows (search, cart, the stock rollback,
double-click checkout, cancel, admin CRUD, mobile layout) were also driven through a real browser against both
modes; that script is not part of the repo because it needs Playwright.

## Deploying the demo

Import the repo on Vercel with **Root Directory** `web` and no environment variables. That gives demo
mode, which needs no backend.

## Layout

```
app/                 pages: shop, product, cart, orders, order, login, admin/products
app/api/             real-mode proxy (catch-all) and the login/logout routes that manage the cookie
components/          UI pieces (header, cards, forms, dialogs)
lib/api.ts           the browser API client; switches between the proxy and the demo
lib/demo/            the in-browser shop and its tests
lib/server/          server-only helpers for the proxy
scripts/             demo-vs-real conformance check
```
