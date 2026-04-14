"use client";

import { useEffect, useRef, useState } from "react";
import { IconSearch } from "@tabler/icons-react";

import { cn } from "@/lib/utils";
import { GLASS_PILL_CLASS, GLASS_PILL_HEIGHT } from "@/components/glass-pill";

const COLLAPSED_WIDTH = GLASS_PILL_HEIGHT;
const EXPAND_EASING = "cubic-bezier(0.32, 0.72, 0.24, 1)"; // Apple-style soft-out

// Glass pill chrome from glass-pill.ts, plus overflow clipping for the
// expand animation and h-[42px] for the canonical height.
const baseClass = cn(
  GLASS_PILL_CLASS,
  "relative shrink-0 h-[42px] overflow-hidden hover:text-foreground",
);

interface ExpandingSearchProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  /** Width (px) of the expanded input. Defaults to 240. */
  expandedWidth?: number;
  className?: string;
}

/**
 * A search control that renders as a round icon button by default and
 * expands horizontally into a full input field. The animation is purely
 * geometric: the outer pill animates its width, and `overflow-hidden`
 * reveals the input as the clip widens. The input never moves - it's
 * always in its final position, hidden only by the clipping edge.
 */
export function ExpandingSearch({
  value,
  onChange,
  placeholder = "Search…",
  expandedWidth = 240,
  className = "",
}: ExpandingSearchProps) {
  const [expanded, setExpanded] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const rootRef = useRef<HTMLDivElement>(null);

  // Keep it expanded as long as there's text. Search chrome shouldn't
  // collapse mid-typing, and it shouldn't hide an active filter value.
  const shouldShow = expanded || value.length > 0;

  useEffect(() => {
    if (shouldShow) inputRef.current?.focus();
  }, [shouldShow]);

  useEffect(() => {
    if (!expanded) return;
    const onPointerDown = (e: MouseEvent) => {
      const target = e.target as Node | null;
      if (!target) return;
      if (rootRef.current?.contains(target)) return;
      if (!value) setExpanded(false);
    };
    document.addEventListener("mousedown", onPointerDown);
    return () => document.removeEventListener("mousedown", onPointerDown);
  }, [expanded, value]);

  return (
    <div
      ref={rootRef}
      className={cn(
        baseClass,
        !shouldShow && "cursor-pointer hover:bg-foreground/[0.07] dark:hover:bg-white/[0.08]",
        className,
      )}
      style={{
        width: shouldShow ? expandedWidth : COLLAPSED_WIDTH,
        transition: `width 320ms ${EXPAND_EASING}, background-color 200ms ease-out`,
      }}
      onClick={() => { if (!shouldShow) setExpanded(true); }}
      role={shouldShow ? undefined : "button"}
      aria-label={shouldShow ? undefined : "Search"}
      tabIndex={shouldShow ? -1 : 0}
      onKeyDown={(e) => {
        if (!shouldShow && (e.key === "Enter" || e.key === " ")) {
          e.preventDefault();
          setExpanded(true);
        }
      }}
    >
      {/* Icon: absolutely positioned in a 40×40 square that exactly
          matches the pill's padding-box (interior after the 1px
          border). This makes the icon position totally independent of
          outer width - it always sits at the geometric center (21, 21)
          of the 42px pill whether collapsed, expanded, or mid-animation. */}
      <span className="absolute top-0 left-0 w-10 h-10 flex items-center justify-center pointer-events-none">
        <IconSearch size={16} stroke={2.4} className="translate-y-[0.5px]" />
      </span>
      {/* Input: filling the padding box from x=40 to the right edge.
          When outer width is 42px (interior 40px), left:40 right:0
          collapses to 0 width - fully behind the clipping edge. */}
      <input
        ref={inputRef}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Escape") {
            onChange("");
            setExpanded(false);
            (e.target as HTMLInputElement).blur();
          }
        }}
        onBlur={() => { if (!value) setExpanded(false); }}
        placeholder={placeholder}
        tabIndex={shouldShow ? 0 : -1}
        aria-hidden={!shouldShow}
        className={cn(
          "absolute top-0 bottom-0 left-10 right-0 pl-1 pr-4 bg-transparent text-sm leading-10 text-foreground/85 placeholder:text-muted-foreground/30 focus:outline-none",
          !shouldShow && "pointer-events-none",
        )}
      />
    </div>
  );
}
