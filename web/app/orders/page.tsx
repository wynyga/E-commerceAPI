"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";

import OrderStatusBadge from "@/components/OrderStatusBadge";
import RequireAuth from "@/components/RequireAuth";
import { Card, EmptyState, ErrorState, PageHeader, Pagination, SkeletonRows } from "@/components/ui";
import { api } from "@/lib/api";
import { formatDate, orderNumber, rupiah } from "@/lib/format";

export default function OrdersPage() {
  return (
    <RequireAuth>
      <Orders />
    </RequireAuth>
  );
}

function Orders() {
  const [page, setPage] = useState(1);
  const q = useQuery({ queryKey: ["orders", "list", page], queryFn: () => api.orders.list(page), placeholderData: keepPreviousData });

  return (
    <>
      <PageHeader title="Your orders" subtitle="Newest first. An order keeps the names and prices it was placed with." />
      <Card>
        {q.isPending ? (
          <SkeletonRows rows={4} />
        ) : q.isError ? (
          <ErrorState message={q.error.message} onRetry={() => q.refetch()} />
        ) : q.data.items.length === 0 ? (
          <EmptyState
            title="No orders yet"
            hint="When you check out, your order shows up here."
            action={
              <Link href="/" className="inline-block rounded-lg bg-accent px-3.5 py-2 text-sm font-medium text-accent-ink hover:opacity-90">
                Browse the shop
              </Link>
            }
          />
        ) : (
          <>
            <ul className="divide-y divide-line">
              {q.data.items.map((o) => {
                return (
                  <li key={o.id}>
                    <Link href={`/orders/${o.id}`} className="flex flex-wrap items-center gap-x-6 gap-y-2 px-5 py-4 hover:bg-surface2">
                      <div className="min-w-32">
                        <p className="font-medium tabular-nums">Order {orderNumber(o.id)}</p>
                        <p className="text-xs text-muted">{formatDate(o.created_at)}</p>
                      </div>
                      <p className="min-w-0 flex-1 truncate text-sm text-muted">
                        {o.items.map((i) => i.name).join(", ")}
                      </p>
                      <OrderStatusBadge status={o.status} />
                      <p className="w-32 text-right font-semibold tabular-nums">{rupiah(o.total)}</p>
                    </Link>
                  </li>
                );
              })}
            </ul>
            <Pagination
              meta={{ page: q.data.page, pageSize: q.data.limit, total: q.data.total, totalPages: q.data.total_pages }}
              onPage={setPage}
            />
          </>
        )}
      </Card>
    </>
  );
}
