// Shapes returned by the Go services (snake_case, as on the wire).

export type Role = "customer" | "admin";

export type User = { id: number; email: string; role: Role; created_at: string };

export type Product = {
  id: number;
  name: string;
  description: string;
  /** Whole rupiah. */
  price: number;
  stock: number;
  /** Demo catalogue only: a photo under /public. The Go product service has no image field yet. */
  image?: string;
  created_at: string;
  updated_at: string;
};

export type ProductInput = { name: string; description: string; price: number; stock: number };

export type Paged<T> = { items: T[]; page: number; limit: number; total: number; total_pages: number };

export type ProductSort = "name" | "price" | "-price" | "newest";

export type ProductQuery = { q?: string; sort?: ProductSort; page?: number; limit?: number };

export type CartItem = { id: number; cart_id: number; product_id: number; quantity: number };

export type Cart = { id: number; user_id: number; items: CartItem[] };

export type OrderStatus = "pending" | "cancelled";

export type OrderItem = { id: number; product_id: number; name: string; unit_price: number; quantity: number };

export type Order = {
  id: number;
  user_id: number;
  status: OrderStatus;
  total: number;
  items: OrderItem[];
  created_at: string;
};
