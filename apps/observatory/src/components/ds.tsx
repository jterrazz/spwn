/**
 * Design system primitives — the diagnostics panel aesthetic.
 *
 * Dark background, mono font, uppercase labels with wide tracking,
 * bold metric values, accent bars on section headers, thin separators.
 * No cards, no glass, no rounded wrappers — raw on dark.
 */

import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

// ── Section Header ──────────────────────────────────────────────────
// Left accent bar + bold uppercase mono label.
//
//   ┃ IDENTITY

interface SectionHeaderProps {
  children: ReactNode;
  className?: string;
}

export function SectionHeader({ children, className }: SectionHeaderProps) {
  return (
    <p className={cn(
      "text-xs font-mono font-bold uppercase tracking-[0.04em] text-foreground/90 mb-3 flex items-center gap-2",
      className,
    )}>
      <span className="w-0.5 h-4 bg-foreground/90 rounded-full shrink-0" />
      {children}
    </p>
  );
}

// ── Section Label ───────────────────────────────────────────────────
// Bold uppercase white — used for content sections within a panel.
// No accent bar (that's reserved for SectionHeader / main titles).
//
//   SERVICES

interface SectionLabelProps {
  children: ReactNode;
  className?: string;
}

export function SectionLabel({ children, className }: SectionLabelProps) {
  return (
    <p className={cn(
      "text-[10px] font-mono font-bold uppercase tracking-[0.04em] text-foreground/90 mb-3",
      className,
    )}>
      {children}
    </p>
  );
}

// ── Sub Label ───────────────────────────────────────────────────────
// Smaller label without accent bar — used for nested fields.
//
//   PURPOSE

interface SubLabelProps {
  children: ReactNode;
  className?: string;
}

export function SubLabel({ children, className }: SubLabelProps) {
  return (
    <p className={cn(
      "text-[9px] font-mono uppercase tracking-[0.03em] text-muted-foreground/35",
      className,
    )}>
      {children}
    </p>
  );
}

// ── Separator ───────────────────────────────────────────────────────
// Thin horizontal line between sections.

export function Separator({ className }: { className?: string }) {
  return <div className={cn("h-px bg-white/[0.06]", className)} />;
}

// ── Metric Grid ─────────────────────────────────────────────────────
// 2-column grid of large bold mono values with tiny labels.
//
//   FILES        LAYERS
//   42           6

interface MetricItem {
  label: string;
  value: string | number;
}

interface MetricGridProps {
  items: MetricItem[];
  columns?: 2 | 3 | 4;
  className?: string;
}

export function MetricGrid({ items, columns = 2, className }: MetricGridProps) {
  const colClass = columns === 2 ? "grid-cols-2" : columns === 3 ? "grid-cols-3" : "grid-cols-4";
  return (
    <div className={cn(`grid ${colClass} gap-x-6 gap-y-3`, className)}>
      {items.map(({ label, value }) => (
        <div key={label}>
          <SubLabel>{label}</SubLabel>
          <p className="text-xl font-mono font-bold text-foreground/90 mt-0.5">{value}</p>
        </div>
      ))}
    </div>
  );
}

// ── Key Value ────────────────────────────────────────────────────────
// Label left, value right. Optional status dot.
//
//   Engine                    claude-code
//   Status                  ● Running

interface KeyValueProps {
  label: string;
  value: string | number;
  dot?: string; // Tailwind bg color class for status dot, e.g. "bg-green-500"
  className?: string;
}

export function KeyValue({ label, value, dot, className }: KeyValueProps) {
  return (
    <div className={cn("flex items-center justify-between", className)}>
      <SubLabel>{label}</SubLabel>
      <div className="flex items-center gap-1.5">
        {dot && <span className={`w-1.5 h-1.5 rounded-full ${dot}`} />}
        <span className="text-xs font-mono font-medium text-foreground/80">{value}</span>
      </div>
    </div>
  );
}

// ── Item List ────────────────────────────────────────────────────────
// Dot-prefixed mono list — services / deployment history style.
//
//   ● Auth Service              Running · 99.99%
//   ● Database                  Degraded · 92.11%

interface ItemListEntry {
  name: string;
  detail?: string;
  href?: string;
}

interface ItemListProps {
  items: ItemListEntry[];
  className?: string;
}

export function ItemList({ items, className }: ItemListProps) {
  return (
    <div className={cn("space-y-2", className)}>
      {items.map((item) => {
        const content = (
          <>
            <span className="w-[6px] h-[6px] rounded-full bg-foreground/80 shrink-0" />
            <span className="text-xs font-mono font-bold text-foreground/90 flex-1 truncate">{item.name}</span>
            {item.detail && (
              <span className="text-[10px] font-mono text-foreground/80 shrink-0">{item.detail}</span>
            )}
          </>
        );
        if (item.href) {
          return (
            <a key={item.name} href={item.href} className="group flex items-center gap-2 hover:text-foreground transition-colors">
              {content}
            </a>
          );
        }
        return (
          <div key={item.name} className="group flex items-center gap-2">
            {content}
          </div>
        );
      })}
    </div>
  );
}

// ── Progress Bar ────────────────────────────────────────────────────
// Label left, percentage right, flat bar below.
//
//   AGENTS ALIVE                              100%
//   ██████████████████████████████████████░░░░░░

interface ProgressBarProps {
  label: string;
  value: number;   // 0–100
  className?: string;
}

export function ProgressBar({ label, value, className }: ProgressBarProps) {
  const clamped = Math.max(0, Math.min(100, value));
  return (
    <div className={cn("", className)}>
      <div className="flex items-center justify-between mb-2">
        <SubLabel>{label}</SubLabel>
        <span className="text-xs font-mono font-bold text-foreground/90">{Math.round(clamped)}%</span>
      </div>
      <div className="h-2 w-full flex rounded-[1px] overflow-hidden">
        <div className="bg-foreground/90 transition-all duration-500" style={{ width: `${clamped}%` }} />
        <div className="bg-foreground/15 flex-1" />
      </div>
    </div>
  );
}

// ── Status Dot ──────────────────────────────────────────────────────
// Colored circle indicating running/idle/stopped/error state.

const STATUS_DOT_COLORS: Record<string, string> = {
  running: "bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.6)]",
  idle: "bg-amber-400 shadow-[0_0_6px_rgba(251,191,36,0.5)]",
  waiting: "bg-amber-400 animate-pulse",
  sleeping: "bg-purple-400",
  stopped: "bg-zinc-500/40",
  error: "bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.6)]",
  creating: "bg-blue-400 shadow-[0_0_6px_rgba(96,165,250,0.5)]",
};

interface StatusDotProps {
  status: string;
  size?: "sm" | "md";
  className?: string;
}

export function StatusDot({ status, size = "sm", className }: StatusDotProps) {
  const sizeClass = size === "md" ? "w-2.5 h-2.5" : "w-1.5 h-1.5";
  return (
    <span className={cn(`rounded-full ${sizeClass} ${STATUS_DOT_COLORS[status] ?? STATUS_DOT_COLORS.stopped}`, className)} />
  );
}

// ── Data Table ─────────────────────────────────────────────────────
// Notion/Vercel-style table: thin border, header row, clean data rows.
// Each column is defined by a key, label, and optional render function.
//
//   ┌─────────────────────────────────────────────┐
//   │ NAME           ROLE        STATUS            │
//   ├─────────────────────────────────────────────┤
//   │ QA Eng         citizen     ● running         │
//   │ Coder          governor    ● idle            │
//   └─────────────────────────────────────────────┘

interface DataTableColumn<T> {
  key: string;
  label: string;
  /** Column width as CSS grid value, e.g. "1fr", "80px" */
  width?: string;
  render: (row: T) => ReactNode;
}

interface DataTableProps<T> {
  columns: DataTableColumn<T>[];
  rows: T[];
  /** Unique key extractor */
  rowKey: (row: T) => string;
  /** If provided, rows become clickable links */
  rowHref?: (row: T) => string;
  /** Show arrow on hover for clickable rows */
  showArrow?: boolean;
  className?: string;
  emptyText?: string;
}

export function DataTable<T>({ columns, rows, rowKey, rowHref, showArrow = true, className, emptyText }: DataTableProps<T>) {
  const gridCols = columns.map((c) => c.width ?? "1fr").join(" ");
  const gridTemplate = showArrow && rowHref ? `${gridCols} 28px` : gridCols;

  if (rows.length === 0 && emptyText) {
    return <p className="text-xs text-muted-foreground/30 font-mono py-4">{emptyText}</p>;
  }

  return (
    <div className={cn("border border-white/[0.06] rounded-lg overflow-hidden", className)}>
      {/* Header */}
      <div
        className="items-center gap-3 px-3 py-2 border-b border-white/[0.06] bg-white/[0.02] hidden sm:grid"
        style={{ gridTemplateColumns: gridTemplate }}
      >
        {columns.map((col) => (
          <SubLabel key={col.key} className="mb-0">{col.label}</SubLabel>
        ))}
        {showArrow && rowHref && <span />}
      </div>
      {/* Rows */}
      {rows.map((row, i) => {
        const content = (
          <>
            {columns.map((col) => (
              <span key={col.key} className="min-w-0">{col.render(row)}</span>
            ))}
            {showArrow && rowHref && (
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground/15 group-hover:text-foreground/50 transition-colors shrink-0">
                <path d="M5 12h14" /><path d="m12 5 7 7-7 7" />
              </svg>
            )}
          </>
        );

        const cls = `grid items-center gap-3 px-3 py-2.5 group transition-colors ${
          i < rows.length - 1 ? "border-b border-white/[0.04]" : ""
        } ${rowHref ? "hover:bg-white/[0.03] cursor-pointer" : ""}`;

        if (rowHref) {
          return (
            <a key={rowKey(row)} href={rowHref(row)} className={cls} style={{ gridTemplateColumns: gridTemplate }}>
              {content}
            </a>
          );
        }
        return (
          <div key={rowKey(row)} className={cls} style={{ gridTemplateColumns: gridTemplate }}>
            {content}
          </div>
        );
      })}
    </div>
  );
}
