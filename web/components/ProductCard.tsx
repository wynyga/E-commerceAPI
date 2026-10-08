"use client";

import Link from "next/link";

import { rupiah } from "@/lib/format";
import { useAddToCart } from "@/lib/shop-hooks";
import type { Product } from "@/lib/types";
import ProductTile from "./ProductTile";
import StockBadge from "./StockBadge";
import { Button } from "./ui";

export default function ProductCard({ product }: { product: Product }) {
  const { add, pending } = useAddToCart();
  const soldOut = product.stock <= 0;

  return (
    <article className="flex flex-col overflow-hidden rounded-2xl border border-line bg-surface">
      <Link href={`/products/${product.id}`} className="block" tabIndex={-1} aria-hidden="true">
        <ProductTile product={product} />
      </Link>
      <div className="flex flex-1 flex-col gap-2 p-4">
        <h2 className="font-display text-lg leading-snug">
          <Link href={`/products/${product.id}`} className="hover:underline">
            {product.name}
          </Link>
        </h2>
        <div className="flex items-center justify-between gap-2">
          <p className="text-base font-semibold tabular-nums">{rupiah(product.price)}</p>
          <StockBadge stock={product.stock} />
        </div>
        <Button
          variant="primary"
          size="sm"
          className="mt-auto"
          disabled={soldOut}
          loading={pending}
          onClick={() => add(product)}
          aria-label={`Add ${product.name} to cart`}
        >
          {soldOut ? "Sold out" : "Add to cart"}
        </Button>
      </div>
    </article>
  );
}
