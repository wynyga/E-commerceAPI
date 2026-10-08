/** Generated artwork for a product: a gradient whose hue comes from the id, plus the name's initial. */
export default function ProductTile({ id, name, large = false, className = "" }: { id: number; name: string; large?: boolean; className?: string }) {
  const hue = (id * 47 + 18) % 360;
  return (
    <div className={`tile ${large ? "tile-lg" : ""} ${className}`} style={{ "--h": hue } as React.CSSProperties} aria-hidden="true">
      <span>{name.trim().charAt(0).toUpperCase()}</span>
    </div>
  );
}
