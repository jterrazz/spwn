import type { ReactNode } from 'react';

import { GLASS_PILL_CLASS } from '@/components/glass-pill';
import { cn } from '@/styles/class-names';

// Icon-only round variant and icon+text pill variant share the same base
// Glass chrome. See components/glass-pill.ts for the canonical styling.
const baseClass = cn(
    GLASS_PILL_CLASS,
    'flex shrink-0 items-center justify-center gap-2 transition-colors',
    'hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50',
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
 * ActionButton renders a consistent header action - either as a compact
 * round icon-only button, or as a pill with icon + text. Both share the
 * exact same base styling so swapping between them is cosmetic only.
 */
export function ActionButton({
    icon,
    label,
    onClick,
    compact,
    danger,
    className,
    disabled,
}: ActionButtonProps) {
    if (compact) {
        return (
            <button
                aria-label={label}
                className={cn(
                    GLASS_PILL_CLASS,
                    'group/btn relative h-[42px] shrink-0 overflow-hidden',
                    danger
                        ? 'text-red-400/60 hover:border-red-500/25 hover:text-red-400'
                        : 'hover:text-foreground',
                    'disabled:cursor-not-allowed disabled:opacity-50',
                    className,
                )}
                disabled={disabled}
                onClick={onClick}
                onMouseEnter={(e) => {
                    e.currentTarget.style.width = `${42 + label.length * 8 + 20}px`;
                }}
                onMouseLeave={(e) => {
                    e.currentTarget.style.width = '42px';
                }}
                style={{
                    width: 42,
                    transition: 'width 320ms cubic-bezier(0.32, 0.72, 0.24, 1)',
                }}
                type="button"
            >
                <span className="pointer-events-none absolute top-0 left-0 flex h-10 w-10 items-center justify-center">
                    {icon}
                </span>
                <span className="absolute top-0 bottom-0 left-[42px] flex items-center pr-4 text-sm whitespace-nowrap">
                    {label}
                </span>
            </button>
        );
    }
    return (
        <button
            aria-label={label}
            className={cn(
                baseClass,
                'h-[42px] px-5 text-sm',
                danger && 'hover:border-red-500/25 hover:text-red-400',
                className,
            )}
            disabled={disabled}
            onClick={onClick}
            type="button"
        >
            {icon}
            <span>{label}</span>
        </button>
    );
}
