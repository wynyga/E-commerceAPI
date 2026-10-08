import { Badge } from "./ui";

/** Status is always icon + words, never colour alone. */
export default function StockBadge({ stock }: { stock: number }) {
  if (stock <= 0) return <Badge tone="danger">✕ Sold out</Badge>;
  if (stock <= 5) return <Badge tone="warn">! Only {stock} left</Badge>;
  return <Badge tone="ok">✓ In stock</Badge>;
}
