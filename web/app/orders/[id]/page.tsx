"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";

import OrderStatusBadge from "@/components/OrderStatusBadge";
import RequireAuth from "@/components/RequireAuth";
import { Button, Card, ConfirmDialog, EmptyState, ErrorState, PageHeader, SkeletonRows } from "@/components/ui";
import { api, ApiError } from "@/lib/api";
import { formatDate, orderNumber, rupiah } from "@/lib/format";
import { useAction } from "@/lib/hooks";

export default function OrderPage() {
  return (
    <RequireAuth>
      <OrderDetail />
    </RequireAuth>
  );
}

function OrderDetail() {
  const { id } = useParams<{ id: string }>();
  const orderId = Number(id);
  const [confirming, setConfirming] = useState(false);

  const q = useQuery({ queryKey: ["orders", "one", orderId], queryFn: () => api.orders.get(orderId), enabled: Number.isInteger(orderId) && orderId > 0 });
  const cancel = useAction(api.orders.cancel, {
    success: "Order cancelled and the stock returned",
    invalidate: ["orders", "products", "product"],
  });

  if (q.isPending && q.fetchStatus !== "idle") {
    return (
      <Card>
        <SkeletonRows rows={3} />
      </Card>
    );
  }
  if (q.isError && !(q.error instanceof ApiError && q.error.status === 404)) {
    return (
      <Card>
        <ErrorState message={q.error.message} onRetry={() => q.refetch()} />
      </Card>
    );
  }
  const o = q.data;
  if (!o) {
    return (
      <Card>
        <EmptyState
          title="Order not found"
          hint="It doesn’t exist, or it belongs to another account."
          action={
            <Link href="/orders" className="text-sm underline">
              Back to your orders
            </Link>
          }
        />
      </Card>
    );
  }

  return (
    <>
      <Link href="/orders" className="mb-5 inline-block text-sm text-muted hover:text-ink">
        ← All orders
      </Link>
      <PageHeader
        title={`Order ${orderNumber(o.id)}`}
        subtitle={formatDate(o.created_at)}
        actions={
          <>
            <OrderStatusBadge status={o.status} />
            {o.status === "pending" && (
              <Button variant="danger" size="sm" onClick={() => setConfirming(true)}>
                Cancel order
              </Button>
            )}
          </>
        }
      />

      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-line text-xs text-muted">
              <tr>
                <th scope="col" className="px-5 py-3 font-medium">
                  Item
                </th>
                <th scope="col" className="px-3 py-3 text-right font-medium">
                  Price
                </th>
                <th scope="col" className="px-3 py-3 text-right font-medium">
                  Qty
                </th>
                <th scope="col" className="px-5 py-3 text-right font-medium">
                  Subtotal
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {o.items.map((i) => (
                <tr key={i.id}>
                  <td className="px-5 py-3 font-medium">{i.name}</td>
                  <td className="px-3 py-3 text-right tabular-nums">{rupiah(i.unit_price)}</td>
                  <td className="px-3 py-3 text-right tabular-nums">{i.quantity}</td>
                  <td className="px-5 py-3 text-right tabular-nums">{rupiah(i.unit_price * i.quantity)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot className="border-t border-line">
              <tr>
                <th scope="row" colSpan={3} className="px-5 py-3 text-right font-medium text-muted">
                  Total
                </th>
                <td className="px-5 py-3 text-right font-display text-xl tabular-nums">{rupiah(o.total)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </Card>

      <p className="mt-4 text-sm text-muted">
        {o.status === "cancelled"
          ? "This order was cancelled and its stock went back on the shelf."
          : "Prices above are the ones at the time of purchase, whatever the catalogue says today. Cancelling returns the stock exactly once."}
      </p>

      <ConfirmDialog
        open={confirming}
        title="Cancel this order?"
        message="The items go back on the shelf for other shoppers. This can’t be undone."
        confirmLabel="Cancel order"
        onClose={() => setConfirming(false)}
        onConfirm={async () => {
          const res = await cancel.run(o.id);
          if (res.ok) setConfirming(false);
        }}
      />
    </>
  );
}
