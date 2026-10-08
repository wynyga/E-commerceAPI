"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useState } from "react";

import ProductForm from "@/components/ProductForm";
import RequireAuth from "@/components/RequireAuth";
import StockBadge from "@/components/StockBadge";
import { Button, Card, ConfirmDialog, EmptyState, ErrorState, Modal, PageHeader, Pagination, SkeletonRows } from "@/components/ui";
import { api } from "@/lib/api";
import { rupiah } from "@/lib/format";
import { useAction } from "@/lib/hooks";
import type { Product } from "@/lib/types";

export default function AdminProductsPage() {
  return (
    <RequireAuth role="admin">
      <AdminProducts />
    </RequireAuth>
  );
}

function AdminProducts() {
  const [page, setPage] = useState(1);
  const [editing, setEditing] = useState<Product | "new" | null>(null);
  const [deleting, setDeleting] = useState<Product | null>(null);

  const q = useQuery({
    queryKey: ["products", "admin", page],
    queryFn: () => api.products.list({ sort: "newest", page, limit: 20 }),
    placeholderData: keepPreviousData,
  });
  const remove = useAction(api.products.remove, { success: "Product deleted", invalidate: ["products", "product"] });

  return (
    <>
      <PageHeader
        title="Manage products"
        subtitle="Only admins can change the catalogue. Customers get a 403 from the product service."
        actions={
          <Button variant="primary" onClick={() => setEditing("new")}>
            New product
          </Button>
        }
      />

      <Card>
        {q.isPending ? (
          <SkeletonRows rows={6} />
        ) : q.isError ? (
          <ErrorState message={q.error.message} onRetry={() => q.refetch()} />
        ) : q.data.items.length === 0 ? (
          <EmptyState title="No products" hint="Create the first one." />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="border-b border-line text-xs text-muted">
                  <tr>
                    <th scope="col" className="px-5 py-3 font-medium">
                      Product
                    </th>
                    <th scope="col" className="px-3 py-3 text-right font-medium">
                      Price
                    </th>
                    <th scope="col" className="px-3 py-3 font-medium">
                      Stock
                    </th>
                    <th scope="col" className="px-5 py-3 text-right font-medium">
                      <span className="sr-only">Actions</span>
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-line">
                  {q.data.items.map((p) => (
                    <tr key={p.id}>
                      <td className="px-5 py-3">
                        <Link href={`/products/${p.id}`} className="font-medium hover:underline">
                          {p.name}
                        </Link>
                      </td>
                      <td className="px-3 py-3 text-right whitespace-nowrap tabular-nums">{rupiah(p.price)}</td>
                      <td className="px-3 py-3 whitespace-nowrap">
                        <span className="mr-2 tabular-nums">{p.stock}</span>
                        <StockBadge stock={p.stock} />
                      </td>
                      <td className="px-5 py-3 text-right whitespace-nowrap">
                        <Button size="sm" variant="ghost" onClick={() => setEditing(p)} aria-label={`Edit ${p.name}`}>
                          Edit
                        </Button>
                        <Button size="sm" variant="ghost" onClick={() => setDeleting(p)} aria-label={`Delete ${p.name}`}>
                          Delete
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pagination
              meta={{ page: q.data.page, pageSize: q.data.limit, total: q.data.total, totalPages: q.data.total_pages }}
              onPage={setPage}
            />
          </>
        )}
      </Card>

      <Modal open={editing !== null} onClose={() => setEditing(null)} title={editing === "new" ? "New product" : "Edit product"}>
        {editing !== null && (
          <ProductForm product={editing === "new" ? undefined : editing} onDone={() => setEditing(null)} onCancel={() => setEditing(null)} />
        )}
      </Modal>

      <ConfirmDialog
        open={deleting !== null}
        title="Delete this product?"
        message={
          <>
            <strong>{deleting?.name}</strong> disappears from the shop. Orders that already contain it keep their copy of its name and price.
          </>
        }
        confirmLabel="Delete"
        onClose={() => setDeleting(null)}
        onConfirm={async () => {
          if (!deleting) return;
          const res = await remove.run(deleting.id);
          if (res.ok) setDeleting(null);
        }}
      />
    </>
  );
}
