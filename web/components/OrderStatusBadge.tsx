import type { OrderStatus } from "@/lib/types";
import { Badge } from "./ui";

/** Icon + word, never colour alone. */
export default function OrderStatusBadge({ status }: { status: OrderStatus }) {
  return status === "cancelled" ? <Badge tone="danger">✕ Cancelled</Badge> : <Badge tone="neutral">◷ Pending</Badge>;
}
