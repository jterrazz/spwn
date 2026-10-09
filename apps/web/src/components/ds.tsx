import type { ReactNode } from 'react';

import { cn } from '../styles/class-names';

interface SectionHeaderProps {
    children: ReactNode;
    className?: string;
}

export function SectionHeader({ children, className }: SectionHeaderProps) {
    return (
        <p
            className={cn(
                'text-foreground/90 mb-3 flex items-center gap-2 font-mono text-xs font-bold tracking-[0.04em] uppercase',
                className,
            )}
        >
            <span className="bg-foreground/90 h-4 w-0.5 shrink-0 rounded-full" />
            {children}
        </p>
    );
}

interface SectionLabelProps {
    children: ReactNode;
    className?: string;
}

export function SectionLabel({ children, className }: SectionLabelProps) {
    return (
        <p
            className={cn(
                'text-foreground/90 mb-3 font-mono text-[10px] font-bold tracking-[0.04em] uppercase',
                className,
            )}
        >
            {children}
        </p>
    );
}

interface SubLabelProps {
    children: ReactNode;
    className?: string;
}

export function SubLabel({ children, className }: SubLabelProps) {
    return (
        <p
            className={cn(
                'text-muted-foreground/35 font-mono text-[9px] tracking-[0.03em] uppercase',
                className,
            )}
        >
            {children}
        </p>
    );
}

export function Separator({ className }: { className?: string }) {
    return <div className={cn('h-px bg-white/[0.06]', className)} />;
}

interface MetricItem {
    label: string;
    value: number | string;
}

interface MetricGridProps {
    items: MetricItem[];
    columns?: 2 | 3 | 4;
    className?: string;
}

export function MetricGrid({ items, columns = 2, className }: MetricGridProps) {
    let colClass: string;
    if (columns === 2) {
        colClass = 'grid-cols-2';
    } else if (columns === 3) {
        colClass = 'grid-cols-3';
    } else {
        colClass = 'grid-cols-4';
    }
    return (
        <div className={cn(`grid ${colClass} gap-x-6 gap-y-3`, className)}>
            {items.map(({ label, value }) => (
                <div key={label}>
                    <SubLabel>{label}</SubLabel>
                    <p className="text-foreground/90 mt-0.5 font-mono text-xl font-bold">{value}</p>
                </div>
            ))}
        </div>
    );
}

interface KeyValueProps {
    label: string;
    value: number | string;
    dot?: string;
    className?: string;
}

export function KeyValue({ label, value, dot, className }: KeyValueProps) {
    return (
        <div className={cn('flex items-center justify-between', className)}>
            <SubLabel>{label}</SubLabel>
            <div className="flex items-center gap-1.5">
                {dot !== undefined && dot !== '' && (
                    <span className={`h-1.5 w-1.5 rounded-full ${dot}`} />
                )}
                <span className="text-foreground/80 font-mono text-xs font-medium">{value}</span>
            </div>
        </div>
    );
}

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
        <div className={cn('space-y-2', className)}>
            {items.map((item) => {
                const content = (
                    <>
                        <span className="bg-foreground/80 h-[6px] w-[6px] shrink-0 rounded-full" />
                        <span className="text-foreground/90 flex-1 truncate font-mono text-xs font-bold">
                            {item.name}
                        </span>
                        {item.detail !== undefined && item.detail !== '' && (
                            <span className="text-foreground/80 shrink-0 font-mono text-[10px]">
                                {item.detail}
                            </span>
                        )}
                    </>
                );
                if (item.href !== undefined && item.href !== '') {
                    return (
                        <a
                            className="group hover:text-foreground flex items-center gap-2 transition-colors"
                            href={item.href}
                            key={item.name}
                        >
                            {content}
                        </a>
                    );
                }
                return (
                    <div className="group flex items-center gap-2" key={item.name}>
                        {content}
                    </div>
                );
            })}
        </div>
    );
}

interface ProgressBarProps {
    label: string;
    value: number;
    className?: string;
}

export function ProgressBar({ label, value, className }: ProgressBarProps) {
    const clamped = Math.max(0, Math.min(100, value));
    return (
        <div className={cn('', className)}>
            <div className="mb-2 flex items-center justify-between">
                <SubLabel>{label}</SubLabel>
                <span className="text-foreground/90 font-mono text-xs font-bold">
                    {Math.round(clamped)}%
                </span>
            </div>
            <div className="flex h-2 w-full overflow-hidden rounded-[1px]">
                <div
                    className="bg-foreground/90 transition-all duration-500"
                    style={{ width: `${clamped}%` }}
                />
                <div className="bg-foreground/15 flex-1" />
            </div>
        </div>
    );
}

const STATUS_DOT_COLORS: Record<string, string> = {
    running: 'bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.6)]',
    idle: 'bg-amber-400 shadow-[0_0_6px_rgba(251,191,36,0.5)]',
    waiting: 'bg-amber-400 animate-pulse',
    sleeping: 'bg-purple-400',
    stopped: 'bg-zinc-500/40',
    error: 'bg-red-500 shadow-[0_0_6px_rgba(239,68,68,0.6)]',
    creating: 'bg-blue-400 shadow-[0_0_6px_rgba(96,165,250,0.5)]',
};

interface StatusDotProps {
    status: string;
    size?: 'md' | 'sm';
    className?: string;
}

export function StatusDot({ status, size = 'sm', className }: StatusDotProps) {
    const sizeClass = size === 'md' ? 'w-2.5 h-2.5' : 'w-1.5 h-1.5';
    return (
        <span
            className={cn(
                `rounded-full ${sizeClass} ${STATUS_DOT_COLORS[status] ?? STATUS_DOT_COLORS.stopped}`,
                className,
            )}
        />
    );
}

interface DataTableColumn<T> {
    key: string;
    label: string;
    width?: string;
    render: (row: T) => ReactNode;
}

interface DataTableProps<T> {
    columns: DataTableColumn<T>[];
    rows: T[];
    rowKey: (row: T) => string;
    rowHref?: (row: T) => string;
    showArrow?: boolean;
    className?: string;
    emptyText?: string;
}

export function DataTable<T>({
    columns,
    rows,
    rowKey,
    rowHref,
    showArrow = true,
    className,
    emptyText,
}: DataTableProps<T>) {
    const gridCols = columns.map((c) => c.width ?? '1fr').join(' ');
    const gridTemplate = showArrow && rowHref ? `${gridCols} 28px` : gridCols;

    if (rows.length === 0 && emptyText !== undefined && emptyText !== '') {
        return <p className="text-muted-foreground/30 py-4 font-mono text-xs">{emptyText}</p>;
    }

    return (
        <div className={cn('overflow-hidden rounded-lg border border-white/[0.06]', className)}>
            <div
                className="hidden items-center gap-3 border-b border-white/[0.06] bg-white/[0.02] px-3 py-2 sm:grid"
                style={{ gridTemplateColumns: gridTemplate }}
            >
                {columns.map((col) => (
                    <SubLabel className="mb-0" key={col.key}>
                        {col.label}
                    </SubLabel>
                ))}
                {showArrow && rowHref && <span />}
            </div>
            {rows.map((row, i) => {
                const content = (
                    <>
                        {columns.map((col) => (
                            <span className="min-w-0" key={col.key}>
                                {col.render(row)}
                            </span>
                        ))}
                        {showArrow && rowHref && (
                            <svg
                                className="text-muted-foreground/15 group-hover:text-foreground/50 shrink-0 transition-colors"
                                fill="none"
                                height="12"
                                stroke="currentColor"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth="2"
                                viewBox="0 0 24 24"
                                width="12"
                            >
                                <path d="M5 12h14" />
                                <path d="m12 5 7 7-7 7" />
                            </svg>
                        )}
                    </>
                );

                const cls = `grid items-center gap-3 px-3 py-2.5 group transition-colors ${
                    i < rows.length - 1 ? 'border-b border-white/[0.04]' : ''
                } ${rowHref ? 'hover:bg-white/[0.03] cursor-pointer' : ''}`;

                if (rowHref) {
                    return (
                        <a
                            className={cls}
                            href={rowHref(row)}
                            key={rowKey(row)}
                            style={{ gridTemplateColumns: gridTemplate }}
                        >
                            {content}
                        </a>
                    );
                }
                return (
                    <div
                        className={cls}
                        key={rowKey(row)}
                        style={{ gridTemplateColumns: gridTemplate }}
                    >
                        {content}
                    </div>
                );
            })}
        </div>
    );
}
