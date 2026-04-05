import type { ReactNode } from "react";

import { cn } from "@/lib/utils";
import { GLASS_PILL_CLASS } from "@/components/glass-pill";

// Icon-only round variant and icon+text pill variant share the same base
// glass chrome. See components/glass-pill.ts for the canonical styling.
const baseClass = cn(
  GLASS_PILL_CLASS,
  "shrink-0 flex items-center justify-center gap-2 transition-colors",
  "hover:text-foreground disabled:opacity-50 disabled:cursor-not-allowed",
);

interface ActionButtonProps {
  icon: ReactNode;
  label: string;
  onClick: () => void;
  /** When true, render as a 42×42 round icon-only button. The label becomes the aria-label + tooltip. */
  compact?: boolean;
  className?: string;
  disabled?: boolean;
}

/**
 * ActionButton renders a consistent header action — either as a compact
 * round icon-only button, or as a pill with icon + text. Both share the
 * exact same base styling so swapping between them is cosmetic only.
 */
export function ActionButton({ icon, label, onClick, compact, className, disabled }: ActionButtonProps) {
  if (compact) {
    return (
      <button
        type="button"
        onClick={onClick}
        aria-label={label}
        title={label}
        disabled={disabled}
        className={cn(baseClass, "h-[42px] w-[42px]", className)}
      >
        {icon}
      </button>
    );
  }
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      disabled={disabled}
      className={cn(baseClass, "h-[42px] px-5 text-sm", className)}
    >
      {icon}
      <span>{label}</span>
    </button>
  );
}
