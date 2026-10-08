"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useState } from "react";

import { useToast } from "@/components/Toast";
import { ApiError } from "./api";

export function useDebounced<T>(value: T, ms = 250): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return debounced;
}

/** Marks cached lists stale after a change, e.g. invalidate("books", "stats"). */
export function useInvalidate() {
  const qc = useQueryClient();
  return useCallback(
    (...roots: string[]) => Promise.all(roots.map((r) => qc.invalidateQueries({ queryKey: [r] }))),
    [qc],
  );
}

export type ActionResult<R> = { ok: true; data: R } | { ok: false; error: ApiError };

type ActionOptions<R> = {
  /** Toast shown on success; may depend on the result. */
  success?: string | ((r: R) => string);
  /** Query roots to refresh afterwards. */
  invalidate?: string[];
  /** Set false when the caller shows the error itself (forms with field errors). */
  toastError?: boolean;
};

/**
 * Wraps a mutation: tracks pending state, toasts the outcome and refreshes the
 * affected queries. `run` never throws; it resolves to { ok, data | error }.
 */
export function useAction<A extends unknown[], R>(fn: (...args: A) => Promise<R>, opts: ActionOptions<R> = {}) {
  const toast = useToast();
  const invalidate = useInvalidate();
  const [pending, setPending] = useState(false);

  const run = useCallback(
    async (...args: A): Promise<ActionResult<R>> => {
      setPending(true);
      try {
        const data = await fn(...args);
        if (opts.success) toast(typeof opts.success === "function" ? opts.success(data) : opts.success);
        if (opts.invalidate?.length) await invalidate(...opts.invalidate);
        return { ok: true, data };
      } catch (e) {
        const error = e instanceof ApiError ? e : new ApiError("Something went wrong", 0);
        if (opts.toastError !== false) toast(error.message, "error");
        return { ok: false, error };
      } finally {
        setPending(false);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [fn, toast, invalidate],
  );

  return { run, pending };
}
