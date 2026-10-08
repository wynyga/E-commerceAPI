"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";

import { useSession } from "@/components/AuthProvider";
import ProductTile from "@/components/ProductTile";
import QuantityStepper from "@/components/QuantityStepper";
import StockBadge from "@/components/StockBadge";
import { Button, Card, EmptyState, ErrorState } from "@/components/ui";
import { api, ApiError } from "@/lib/api";
import { rupiah } from "@/lib/format";
import { useAddToCart } from "@/lib/shop-hooks";

export default function ProductPage() {
  const { id } = useParams<{ id: string }>();
  const { user } = useSession();
  const { add, pending } = useAddToCart();
  const [qty, setQty] = useState(1);

  const productId = Number(id);
  const q = useQuery({
    queryKey: ["product", productId],
    queryFn: () => api.products.get(productId),
    enabled: Number.isInteger(productId) && productId > 0,
    staleTime: 5_000,
  });

  if (q.isPending && q.fetchStatus !== "idle") {
    return <div aria-hidden="true" className="h-96 animate-pulse rounded-2xl bg-line/60" />;
  }
  if (q.isError && q.error instanceof ApiError && q.error.status !== 404) {
    return (
      <Card>
        <ErrorState message={q.error.message} onRetry={() => q.refetch()} />
      </Card>
    );
  }
  const p = q.data;
  if (!p) {
    return (
      <Card>
        <EmptyState
          title="Product not found"
          hint="It may have been removed from the catalogue."
          action={
            <Link href="/" className="text-sm underline">
              Back to the shop
            </Link>
          }
        />
      </Card>
    );
  }

  const max = Math.min(p.stock, 99);
  const amount = Math.min(qty, Math.max(1, max));

  return (
    <>
      <Link href="/" className="mb-5 inline-block text-sm text-muted hover:text-ink">
        ← All products
      </Link>
      <div className="grid gap-8 md:grid-cols-2">
        <ProductTile id={p.id} name={p.name} large className="rounded-2xl border border-line" />
        <div className="flex flex-col gap-4">
          <div>
            <h1 className="font-display text-3xl leading-tight sm:text-4xl">{p.name}</h1>
            <p className="mt-2 text-2xl font-semibold tabular-nums">{rupiah(p.price)}</p>
          </div>
          <StockBadge stock={p.stock} />
          <p className="max-w-prose leading-relaxed text-muted">{p.description || "No description yet."}</p>

          <div className="mt-2 flex flex-wrap items-center gap-3">
            {p.stock > 0 && <QuantityStepper value={amount} max={max} onChange={setQty} />}
            <Button variant="primary" disabled={p.stock <= 0} loading={pending} onClick={() => add(p, amount)}>
              {p.stock <= 0 ? "Sold out" : user ? "Add to cart" : "Sign in to buy"}
            </Button>
            {user?.role === "admin" && (
              <Link href="/admin/products" className="text-sm underline">
                Edit in admin
              </Link>
            )}
          </div>
          <p className="text-xs text-muted">
            Stock is checked when you add to the cart, and reserved for real at checkout, so the last unit goes to whoever checks out first.
          </p>
        </div>
      </div>
    </>
  );
}
