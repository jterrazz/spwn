'use client';

import { IconSearch } from '@tabler/icons-react';
import { useCallback, useEffect, useRef, useState } from 'react';

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

function useFoldOnOutsideClick(
    expanded: boolean,
    value: string,
    root: React.RefObject<HTMLDivElement | null>,
    fold: () => void,
): void {
    useEffect(() => {
        if (!expanded) {
            return;
        }
        const onPointerDown = (event: MouseEvent) => {
            const outside =
                event.target instanceof Node && root.current?.contains(event.target) !== true;
            if (outside && value === '') {
                fold();
            }
        };
        document.addEventListener('mousedown', onPointerDown);
        return () => {
            document.removeEventListener('mousedown', onPointerDown);
        };
    }, [expanded, value, root, fold]);
}

function SearchGlyph() {
    return (
        <span className="pointer-events-none absolute top-0 left-0 flex h-10 w-10 items-center justify-center">
            <IconSearch className="translate-y-[0.5px]" size={16} stroke={2.4} />
        </span>
    );
}

interface SearchInputProps {
    value: string;
    shown: boolean;
    placeholder: string;
    inputRef: React.RefObject<HTMLInputElement | null>;
    onChange: (value: string) => void;
    fold: () => void;
}

function SearchInput({ value, shown, placeholder, inputRef, onChange, fold }: SearchInputProps) {
    return (
        <input
            aria-hidden={!shown}
            className={cn(
                'text-foreground/85 placeholder:text-muted-foreground/30 absolute top-0 right-0 bottom-0 left-10 bg-transparent pr-4 pl-1 text-sm leading-10 focus:outline-none',
                !shown && 'pointer-events-none',
            )}
            onBlur={() => {
                if (value === '') {
                    fold();
                }
            }}
            onChange={(event) => {
                onChange(event.target.value);
            }}
            onKeyDown={(event) => {
                if (event.key === 'Escape') {
                    onChange('');
                    fold();
                    event.currentTarget.blur();
                }
            }}
            placeholder={placeholder}
            ref={inputRef}
            tabIndex={shown ? 0 : -1}
            value={value}
        />
    );
}

/** A search control that renders as a round icon button by default and expands horizontally into a full input field. The animation is purely geometric: the outer pill animates its width, and `overflow-hidden` reveals the input as the clip widens. The input never moves - it's always in its final position, hidden only by the clipping edge. */
export function ExpandingSearch({
    value,
    onChange,
    placeholder = 'Search...',
    expandedWidth = 240,
    className = '',
}: ExpandingSearchProps) {
    const [expanded, setExpanded] = useState(false);
    const inputRef = useRef<HTMLInputElement>(null);
    const rootRef = useRef<HTMLDivElement>(null);
    const shown = expanded || value.length > 0;
    const fold = useCallback(() => {
        setExpanded(false);
    }, []);
    useFoldOnOutsideClick(expanded, value, rootRef, fold);

    useEffect(() => {
        if (shown) {
            inputRef.current?.focus();
        }
    }, [shown]);

    return (
        <div
            className={cn(
                baseClass,
                !shown && 'hover:bg-foreground/[0.07] dark:hover:bg-white/[0.08]',
                className,
            )}
            ref={rootRef}
            style={{
                transition: `width 320ms ${EXPAND_EASING}, background-color 200ms ease-out`,
                width: shown ? expandedWidth : COLLAPSED_WIDTH,
            }}
        >
            {shown && <SearchGlyph />}
            {!shown && (
                <button
                    aria-label="Search"
                    className="absolute inset-0 z-10 cursor-pointer"
                    onClick={() => {
                        setExpanded(true);
                    }}
                    type="button"
                >
                    <SearchGlyph />
                </button>
            )}
            <SearchInput
                fold={fold}
                inputRef={inputRef}
                onChange={onChange}
                placeholder={placeholder}
                shown={shown}
                value={value}
            />
        </div>
    );
}
