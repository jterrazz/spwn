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
  /** Dangerous actions (stop, destroy, delete) tint red on hover. */
  danger?: boolean;
  className?: string;
  disabled?: boolean;
}

/**
 * ActionButton renders a consistent header action — either as a compact
 * round icon-only button, or as a pill with icon + text. Both share the
 * exact same base styling so swapping between them is cosmetic only.
 */
export function ActionButton({ icon, label, onClick, compact, danger, className, disabled }: ActionButtonProps) {
  if (compact) {
    return (
      <button
        type="button"
        onClick={onClick}
        aria-label={label}
        disabled={disabled}
        className={cn(
          GLASS_PILL_CLASS,
          "group/btn shrink-0 relative h-[42px] overflow-hidden",
          danger
            ? "text-red-400/60 hover:text-red-400 hover:border-red-500/25"
            : "hover:text-foreground",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          className,
        )}
        style={{
          width: 42,
          transition: "width 320ms cubic-bezier(0.32, 0.72, 0.24, 1)",
        }}
        onMouseEnter={(e) => { e.currentTarget.style.width = `${42 + label.length * 8 + 20}px`; }}
        onMouseLeave={(e) => { e.currentTarget.style.width = "42px"; }}
      >
        <span className="absolute top-0 left-0 w-10 h-10 flex items-center justify-center pointer-events-none">
          {icon}
        </span>
        <span className="absolute top-0 bottom-0 left-[42px] flex items-center whitespace-nowrap pr-4 text-sm">
          {label}
        </span>
      </button>
    );
  }
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      disabled={disabled}
      className={cn(
        baseClass,
        "h-[42px] px-5 text-sm",
        danger && "hover:text-red-400 hover:border-red-500/25",
        className,
      )}
    >
      {icon}
      <span>{label}</span>
    </button>
  );
}
