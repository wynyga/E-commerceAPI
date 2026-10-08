"use client";

import { usePathname, useRouter } from "next/navigation";
import { useCallback } from "react";

import { useSession } from "@/components/AuthProvider";
import { api } from "./api";
import { useAction } from "./hooks";
import type { Product } from "./types";

/** Adds a product to the cart, or sends a signed-out visitor to sign in first. */
export function useAddToCart() {
  const { user } = useSession();
  const router = useRouter();
  const path = usePathname();
  const act = useAction(api.cart.add, { success: "Added to your cart", invalidate: ["cart"] });

  const add = useCallback(
    async (product: Product, quantity = 1) => {
      if (!user) {
        router.push(`/login?next=${encodeURIComponent(path)}`);
        return { ok: false as const };
      }
      return act.run(product.id, quantity);
    },
    [user, router, path, act],
  );

  return { add, pending: act.pending };
}
