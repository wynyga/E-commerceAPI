"use client";

import { useQueries, useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";

import ProductTile from "@/components/ProductTile";
import QuantityStepper from "@/components/QuantityStepper";
import RequireAuth from "@/components/RequireAuth";
import { Badge, Button, Card, EmptyState, ErrorState, PageHeader, SkeletonRows } from "@/components/ui";
import { api, ApiError, isDemo } from "@/lib/api";
import { rupiah } from "@/lib/format";
import { useAction } from "@/lib/hooks";
import { useToast } from "@/components/Toast";
import type { CartItem, Product } from "@/lib/types";

// crypto.randomUUID only exists in secure contexts (https, localhost); a LAN address over http needs the fallback.
const newKey = () =>
  typeof crypto.randomUUID === "function" ? crypto.randomUUID() : `k-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;

const REFRESH = ["cart", "products", "product", "orders"];

export default function CartPage() {
  return (
    <RequireAuth>
      <Cart />
    </RequireAuth>
  );
}

function Cart() {
  const router = useRouter();
  const toast = useToast();
  const cart = useQuery({ queryKey: ["cart"], queryFn: api.cart.get });
  const items = cart.data?.items ?? [];

  // The cart only holds product ids; names, prices and stock come from the catalogue.
  const products = useQueries({
    queries: items.map((i) => ({
      queryKey: ["product", i.product_id],
      queryFn: () => api.products.get(i.product_id),
      staleTime: 5_000,
      retry: false,
    })),
  });

  const setQuantity = useAction((item: CartItem, quantity: number) => api.cart.setQuantity(item.id, quantity), { invalidate: ["cart"] });
  const remove = useAction((item: CartItem) => api.cart.remove(item.id), { success: "Removed from your cart", invalidate: ["cart"] });
  const clear = useAction(api.cart.clear, { success: "Cart emptied", invalidate: ["cart"] });
  const rival = useAction(api.rival, {
    success: (r) => `Another shopper just bought the last ${r.bought} of "${r.name}"`,
    invalidate: ["products", "product", "cart"],
  });

  // One idempotency key per distinct cart. If the connection drops and the buyer presses
  // Checkout again, the same key is sent and the server answers with the first order.
  const attempt = useRef<{ signature: string; key: string } | null>(null);
  const [problem, setProblem] = useState<{ message: string; conflict: boolean } | null>(null);
  const checkout = useAction(
    (key: string) => api.orders.checkout(key),
    { toastError: false, invalidate: REFRESH },
  );

  async function placeOrder() {
    const signature = items.map((i) => `${i.product_id}:${i.quantity}`).join(",");
    if (attempt.current?.signature !== signature) attempt.current = { signature, key: newKey() };
    setProblem(null);
    const res = await checkout.run(attempt.current.key);
    if (res.ok) {
      attempt.current = null;
      toast("Order placed");
      router.push(`/orders/${res.data.id}`);
    } else {
      const conflict = res.error instanceof ApiError && res.error.status === 409;
      setProblem({ message: res.error.message, conflict });
    }
  }

  if (cart.isPending) {
    return (
      <>
        <PageHeader title="Your cart" />
        <Card>
          <SkeletonRows rows={3} />
        </Card>
      </>
    );
  }
  if (cart.isError) {
    return (
      <Card>
        <ErrorState message={cart.error.message} onRetry={() => cart.refetch()} />
      </Card>
    );
  }
  if (items.length === 0) {
    return (
      <>
        <PageHeader title="Your cart" />
        <Card>
          <EmptyState
            title="Your cart is empty"
            hint="Add something from the shop and it will wait for you here."
            action={
              <Link href="/" className="inline-block rounded-lg bg-accent px-3.5 py-2 text-sm font-medium text-accent-ink hover:opacity-90">
                Browse the shop
              </Link>
            }
          />
        </Card>
      </>
    );
  }

  const lines = items.map((item, i) => ({ item, product: products[i]?.data as Product | undefined, state: products[i] }));
  const loading = lines.some((l) => l.state?.isPending);
  const gone = lines.filter((l) => !l.product && !l.state?.isPending);
  const total = lines.reduce((sum, l) => sum + (l.product ? l.product.price * l.item.quantity : 0), 0);
  const units = items.reduce((n, i) => n + i.quantity, 0);

  return (
    <>
      <PageHeader
        title="Your cart"
        subtitle={`${units} item${units === 1 ? "" : "s"}`}
        actions={
          <Button size="sm" variant="ghost" loading={clear.pending} onClick={() => clear.run()}>
            Empty cart
          </Button>
        }
      />

      <div className="grid items-start gap-6 lg:grid-cols-[1fr_22rem]">
        <Card className="divide-y divide-line">
          {lines.map(({ item, product }) => {
            const short = product && item.quantity > product.stock;
            return (
              <div key={item.id} className="flex gap-4 p-4">
                <div className="w-20 shrink-0 self-start overflow-hidden rounded-lg bg-surface2">
                  {product ? <ProductTile product={product} /> : <div className="aspect-[4/3]" />}
                </div>
                <div className="flex min-w-0 flex-1 flex-col gap-2">
                  <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-1">
                    <div className="min-w-0">
                      {product ? (
                        <Link href={`/products/${product.id}`} className="font-medium hover:underline">
                          {product.name}
                        </Link>
                      ) : (
                        <p className="font-medium text-muted">{loading ? "Loading…" : "Product no longer available"}</p>
                      )}
                      {product && <p className="text-sm text-muted tabular-nums">{rupiah(product.price)} each</p>}
                    </div>
                    {product && <p className="font-semibold tabular-nums">{rupiah(product.price * item.quantity)}</p>}
                  </div>

                  {short && (
                    <p role="alert">
                      <Badge tone="warn">
                        ! {product.stock === 0 ? "Sold out since you added it" : `Only ${product.stock} left, you have ${item.quantity}`}
                      </Badge>
                    </p>
                  )}
                  {!product && !loading && (
                    <p role="alert">
                      <Badge tone="danger">✕ Remove this line to check out</Badge>
                    </p>
                  )}

                  <div className="flex items-center gap-3">
                    {product && (
                      <QuantityStepper
                        value={item.quantity}
                        max={Math.min(99, Math.max(product.stock, item.quantity))}
                        disabled={setQuantity.pending}
                        onChange={(n) => setQuantity.run(item, n)}
                        label={`Quantity of ${product.name}`}
                      />
                    )}
                    <Button size="sm" variant="ghost" disabled={remove.pending} onClick={() => remove.run(item)}>
                      Remove
                    </Button>
                  </div>
                </div>
              </div>
            );
          })}
        </Card>

        <aside className="space-y-4 lg:sticky lg:top-24">
          <Card className="space-y-4 p-5">
            <div className="flex items-baseline justify-between">
              <span className="text-muted">Total</span>
              <span className="font-display text-2xl tabular-nums">{rupiah(total)}</span>
            </div>

            {problem && (
              <div role="alert" className="rounded-lg border border-danger/40 bg-danger-soft p-3 text-sm text-danger">
                <p className="font-medium">{problem.message}</p>
                {problem.conflict && (
                  <p className="mt-1 text-xs">
                    Nothing was ordered, and no stock stays reserved for you: anything already taken for other lines was put back.
                    Adjust the cart and try again.
                  </p>
                )}
              </div>
            )}

            <Button variant="primary" className="w-full" loading={checkout.pending} disabled={loading || gone.length > 0} onClick={placeOrder}>
              Place order
            </Button>
            <p className="text-xs text-muted">
              Payment isn’t part of this demo. Pressing the button twice, or retrying after a dropped connection, never creates two orders.
            </p>
          </Card>

          {isDemo() && (
            <Card className="space-y-3 border-dashed p-5">
              <p className="text-sm font-medium">Demo tool</p>
              <p className="text-xs text-muted">
                Simulate a race: another shopper buys the last units of the highest-numbered product in your cart. Then place the order and watch it fail
                cleanly, with the stock of your other items put back.
              </p>
              <Button size="sm" loading={rival.pending} onClick={() => rival.run()}>
                Another shopper buys it first
              </Button>
            </Card>
          )}

          <details className="rounded-2xl border border-line bg-surface p-4 text-sm">
            <summary className="cursor-pointer font-medium">What happens when you press Place order</summary>
            <ol className="mt-3 list-decimal space-y-1.5 pl-5 text-muted">
              <li>The order service reads your cart and reserves stock line by line, each with one conditional update.</li>
              <li>If any line is short, everything already reserved is given back and nothing is recorded.</li>
              <li>Otherwise the order is saved with the names and prices as they are now, and your cart is emptied.</li>
            </ol>
          </details>
        </aside>
      </div>
    </>
  );
}
