"use client";

import { useState } from "react";

import { api } from "@/lib/api";
import { useAction } from "@/lib/hooks";
import type { Product } from "@/lib/types";
import { Button, ModalActions, TextArea, TextField } from "./ui";

/** Create (no `product`) or edit. Mounted only while its dialog is open, so it always starts fresh. */
export default function ProductForm({ product, onDone, onCancel }: { product?: Product; onDone: () => void; onCancel: () => void }) {
  const [name, setName] = useState(product?.name ?? "");
  const [description, setDescription] = useState(product?.description ?? "");
  const [price, setPrice] = useState(product ? String(product.price) : "");
  const [stock, setStock] = useState(product ? String(product.stock) : "");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState("");

  const save = useAction(
    (input: { name: string; description: string; price: number; stock: number }) => (product ? api.products.update(product.id, input) : api.products.create(input)),
    { toastError: false, success: product ? "Product saved" : "Product created", invalidate: ["products", "product"] },
  );

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setFields({});
    setFormError("");
    // Empty or non-numeric input becomes NaN, which the server rejects with a field message.
    const res = await save.run({ name, description, price: price.trim() === "" ? 0 : Number(price), stock: stock.trim() === "" ? 0 : Number(stock) });
    if (res.ok) return onDone();
    if (res.error.fields) setFields(res.error.fields);
    else setFormError(res.error.message);
  }

  return (
    <form onSubmit={submit} noValidate className="space-y-4">
      <TextField label="Name" value={name} onChange={(e) => setName(e.target.value)} error={fields.name} maxLength={255} autoFocus />
      <TextArea label="Description" value={description} onChange={(e) => setDescription(e.target.value)} error={fields.description} maxLength={5000} />
      <div className="grid gap-4 sm:grid-cols-2">
        <TextField
          label="Price (Rp)"
          type="number"
          inputMode="numeric"
          min={1}
          step={1}
          value={price}
          onChange={(e) => setPrice(e.target.value)}
          error={fields.price}
          hint="Whole rupiah."
        />
        <TextField label="Stock" type="number" inputMode="numeric" min={0} step={1} value={stock} onChange={(e) => setStock(e.target.value)} error={fields.stock} />
      </div>
      {formError && (
        <p role="alert" className="text-sm text-danger">
          {formError}
        </p>
      )}
      <ModalActions>
        <Button onClick={onCancel}>Cancel</Button>
        <Button type="submit" variant="primary" loading={save.pending}>
          {product ? "Save changes" : "Create product"}
        </Button>
      </ModalActions>
    </form>
  );
}
