"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

import { api } from "@/lib/api";
import { useDebounced } from "@/lib/hooks";
import type { ProductSort } from "@/lib/types";
import ProductCard from "./ProductCard";
import { Button, Card, EmptyState, ErrorState, Pagination, SearchInput, Select } from "./ui";

const PAGE_SIZE = 12;

const SORTS: { value: ProductSort; label: string }[] = [
  { value: "name", label: "Name A–Z" },
  { value: "price", label: "Price: low to high" },
  { value: "-price", label: "Price: high to low" },
  { value: "newest", label: "Newest first" },
];

function parseSort(v: string | null): ProductSort {
  return SORTS.some((s) => s.value === v) ? (v as ProductSort) : "name";
}

/** The shop window. Search, sort and page live in the URL, so back/forward and shared links work. */
export default function Catalogue() {
  const params = useSearchParams();
  const router = useRouter();
  const path = usePathname();

  const [text, setText] = useState(params.get("q") ?? "");
  const q = useDebounced(text.trim(), 300);
  const sort = parseSort(params.get("sort"));
  const page = Math.max(1, parseInt(params.get("page") ?? "1", 10) || 1);

  function update(next: { q?: string; sort?: ProductSort; page?: number }) {
    const sp = new URLSearchParams(params.toString());
    const set = (k: string, v: string | undefined, drop: string) => (v === undefined || v === drop ? sp.delete(k) : sp.set(k, v));
    if ("q" in next) set("q", next.q, "");
    if ("sort" in next) set("sort", next.sort, "name");
    set("page", String(next.page ?? 1), "1");
    const s = sp.toString();
    router.replace(s ? `${path}?${s}` : path, { scroll: false });
  }

  // Typing resets to the first page; the URL follows once typing pauses.
  useEffect(() => {
    if (q !== (params.get("q") ?? "")) update({ q });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q]);

  const list = useQuery({
    queryKey: ["products", { q: params.get("q") ?? "", sort, page }],
    queryFn: () => api.products.list({ q: params.get("q") ?? "", sort, page, limit: PAGE_SIZE }),
    placeholderData: keepPreviousData,
  });

  const meta = list.data;
  const filtered = !!params.get("q");

  return (
    <>
      <div className="mb-6 flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
        <div>
          <h1 className="font-display text-3xl leading-tight sm:text-4xl">The shop</h1>
          <p className="mt-1 text-sm text-muted" aria-live="polite">
            {meta ? `${meta.total} product${meta.total === 1 ? "" : "s"}${filtered ? ` matching “${params.get("q")}”` : ""}` : "Loading…"}
          </p>
        </div>
        <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center">
          <div className="sm:w-64">
            <SearchInput value={text} onChange={setText} placeholder="Search products" />
          </div>
          <Select label="Sort products" value={sort} onChange={(v) => update({ sort: v as ProductSort })} options={SORTS} className="w-auto" />
        </div>
      </div>

      {list.isError && !meta ? (
        <Card>
          <ErrorState message={list.error.message} onRetry={() => list.refetch()} />
        </Card>
      ) : !meta ? (
        <div aria-hidden="true" className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {Array.from({ length: 8 }, (_, i) => (
            <div key={i} className="h-72 animate-pulse rounded-2xl bg-line/60" />
          ))}
        </div>
      ) : meta.items.length === 0 ? (
        <Card>
          <EmptyState
            title="Nothing matches"
            hint={filtered ? `No product matches “${params.get("q")}”. Try a shorter search.` : "The catalogue is empty."}
            action={
              filtered ? (
                <Button
                  onClick={() => {
                    setText("");
                    update({ q: "" });
                  }}
                >
                  Clear search
                </Button>
              ) : undefined
            }
          />
        </Card>
      ) : (
        <>
          <div className={`grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 ${list.isPlaceholderData ? "opacity-60" : ""}`}>
            {meta.items.map((p) => (
              <ProductCard key={p.id} product={p} />
            ))}
          </div>
          <Card className="mt-6">
            <Pagination
              meta={{ page: meta.page, pageSize: meta.limit, total: meta.total, totalPages: meta.total_pages }}
              onPage={(n) => update({ page: n })}
            />
          </Card>
        </>
      )}
    </>
  );
}
