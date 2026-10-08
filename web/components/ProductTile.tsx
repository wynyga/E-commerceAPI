import type { Product } from "@/lib/types";

/**
 * A product's picture. The demo catalogue has photos; a product without one (anything an admin adds,
 * or any product from the real services, which have no image field) gets generated artwork instead:
 * a gradient whose hue comes from the id, plus the name's initial.
 */
export default function ProductTile({
  product,
  large = false,
  className = "",
}: {
  product: Pick<Product, "id" | "name" | "image">;
  large?: boolean;
  className?: string;
}) {
  if (product.image) {
    return (
      // A plain <img>: the files are already small, and a static export has no image optimiser.
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={product.image}
        alt=""
        loading="lazy"
        decoding="async"
        className={`aspect-[4/3] w-full bg-surface2 object-cover ${className}`}
      />
    );
  }
  const hue = (product.id * 47 + 18) % 360;
  return (
    <div className={`tile ${large ? "tile-lg" : ""} ${className}`} style={{ "--h": hue } as React.CSSProperties} aria-hidden="true">
      <span>{product.name.trim().charAt(0).toUpperCase()}</span>
    </div>
  );
}
