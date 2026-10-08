/** Whole rupiah with dot separators: 1250000 -> "Rp 1.250.000". Hand-rolled so server and browser always agree. */
export function rupiah(n: number): string {
  return "Rp " + Math.round(n).toString().replace(/\B(?=(\d{3})+(?!\d))/g, ".");
}

const date = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" });
const time = new Intl.DateTimeFormat("en-GB", { hour: "2-digit", minute: "2-digit", hour12: false, timeZone: "UTC" });

export function formatDate(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? "" : `${date.format(d)}, ${time.format(d)} UTC`;
}

export function orderNumber(id: number): string {
  return `#${String(id).padStart(5, "0")}`;
}
