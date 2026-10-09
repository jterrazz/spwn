'use client';

import { IconSearch } from '@tabler/icons-react';
import { useEffect, useRef, useState } from 'react';

import { cn } from '../styles/class-names';
import { GLASS_PILL_CLASS, GLASS_PILL_HEIGHT } from './glass-pill';

const COLLAPSED_WIDTH = GLASS_PILL_HEIGHT;
const EXPAND_EASING = 'cubic-bezier(0.32, 0.72, 0.24, 1)';

const baseClass = cn(
    GLASS_PILL_CLASS,
    'hover:text-foreground relative h-[42px] shrink-0 overflow-hidden',
);

interface ExpandingSearchProps {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    expandedWidth?: number;
    className?: string;
}

/** A search control that renders as a round icon button by default and expands horizontally into a full input field. The animation is purely geometric: the outer pill animates its width, and `overflow-hidden` reveals the input as the clip widens. The input never moves - it's always in its final position, hidden only by the clipping edge. */
export function ExpandingSearch({
    value,
    onChange,
    placeholder = 'Search…',
    expandedWidth = 240,
    className = '',
}: ExpandingSearchProps) {
    const [expanded, setExpanded] = useState(false);
    const inputRef = useRef<HTMLInputElement>(null);
    const rootRef = useRef<HTMLDivElement>(null);

    const shouldShow = expanded || value.length > 0;

    useEffect(() => {
        if (shouldShow) {
            inputRef.current?.focus();
        }
    }, [shouldShow]);

    useEffect(() => {
        if (!expanded) {
            return;
        }
        const onPointerDown = (e: MouseEvent) => {
            const target = e.target as Node | null;
            if (!target) {
                return;
            }
            if (rootRef.current?.contains(target) === true) {
                return;
            }
            if (!value) {
                setExpanded(false);
            }
        };
        document.addEventListener('mousedown', onPointerDown);
        return () => {
            document.removeEventListener('mousedown', onPointerDown);
        };
    }, [expanded, value]);

    return (
        <div
            aria-label={shouldShow ? undefined : 'Search'}
            className={cn(
                baseClass,
                !shouldShow &&
                    'hover:bg-foreground/[0.07] cursor-pointer dark:hover:bg-white/[0.08]',
                className,
            )}
            onClick={() => {
                if (!shouldShow) {
                    setExpanded(true);
                }
            }}
            onKeyDown={(e) => {
                if (!shouldShow && (e.key === 'Enter' || e.key === ' ')) {
                    e.preventDefault();
                    setExpanded(true);
                }
            }}
            ref={rootRef}
            role={shouldShow ? undefined : 'button'}
            style={{
                width: shouldShow ? expandedWidth : COLLAPSED_WIDTH,
                transition: `width 320ms ${EXPAND_EASING}, background-color 200ms ease-out`,
            }}
            tabIndex={shouldShow ? -1 : 0}
        >
            <span className="pointer-events-none absolute top-0 left-0 flex h-10 w-10 items-center justify-center">
                <IconSearch className="translate-y-[0.5px]" size={16} stroke={2.4} />
            </span>
            <input
                aria-hidden={!shouldShow}
                className={cn(
                    'text-foreground/85 placeholder:text-muted-foreground/30 absolute top-0 right-0 bottom-0 left-10 bg-transparent pr-4 pl-1 text-sm leading-10 focus:outline-none',
                    !shouldShow && 'pointer-events-none',
                )}
                onBlur={() => {
                    if (!value) {
                        setExpanded(false);
                    }
                }}
                onChange={(e) => {
                    onChange(e.target.value);
                }}
                onKeyDown={(e) => {
                    if (e.key === 'Escape') {
                        onChange('');
                        setExpanded(false);
                        (e.target as HTMLInputElement).blur();
                    }
                }}
                placeholder={placeholder}
                ref={inputRef}
                tabIndex={shouldShow ? 0 : -1}
                value={value}
            />
        </div>
    );
}
