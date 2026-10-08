"use client";

export default function QuantityStepper({
  value,
  min = 1,
  max,
  disabled,
  onChange,
  label = "Quantity",
}: {
  value: number;
  min?: number;
  max: number;
  disabled?: boolean;
  onChange: (n: number) => void;
  label?: string;
}) {
  const step = "grid size-8 place-items-center rounded-lg text-lg leading-none hover:bg-surface2 disabled:pointer-events-none disabled:opacity-40";
  return (
    <div role="group" aria-label={label} className="inline-flex items-center rounded-lg border border-line bg-surface">
      <button type="button" className={step} disabled={disabled || value <= min} onClick={() => onChange(value - 1)} aria-label="Decrease quantity">
        −
      </button>
      <span className="min-w-8 text-center text-sm tabular-nums" aria-live="polite">
        {value}
      </span>
      <button type="button" className={step} disabled={disabled || value >= max} onClick={() => onChange(value + 1)} aria-label="Increase quantity">
        +
      </button>
    </div>
  );
}
