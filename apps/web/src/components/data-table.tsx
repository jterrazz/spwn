import type { ReactNode } from 'react';

import { cn } from '../styles/class-names';
import { SubLabel } from './ds';

export interface DataTableColumn<T> {
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

function Arrow() {
    return (
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
    );
}

interface RowProps<T> {
    row: T;
    columns: DataTableColumn<T>[];
    template: string;
    last: boolean;
    href: string | undefined;
    arrow: boolean;
}

function Row<T>({ row, columns, template, last, href, arrow }: RowProps<T>) {
    const cells = (
        <>
            {columns.map((column) => (
                <span className="min-w-0" key={column.key}>
                    {column.render(row)}
                </span>
            ))}
            {arrow && <Arrow />}
        </>
    );
    const className = `grid items-center gap-3 px-3 py-2.5 group transition-colors ${last ? '' : 'border-b border-white/[0.04]'} ${href === undefined ? '' : 'hover:bg-white/[0.03] cursor-pointer'}`;
    return href === undefined ? (
        <div className={className} style={{ gridTemplateColumns: template }}>
            {cells}
        </div>
    ) : (
        <a className={className} href={href} style={{ gridTemplateColumns: template }}>
            {cells}
        </a>
    );
}

/** A table of rows on a shared grid, each row a link when the table is given where rows lead. */
export function DataTable<T>({
    columns,
    rows,
    rowKey,
    rowHref,
    showArrow = true,
    className,
    emptyText = '',
}: DataTableProps<T>) {
    const arrow = showArrow && rowHref !== undefined;
    const widths = columns.map((column) => column.width ?? '1fr').join(' ');
    const template = arrow ? `${widths} 28px` : widths;

    if (rows.length === 0 && emptyText !== '') {
        return <p className="text-muted-foreground/30 py-4 font-mono text-xs">{emptyText}</p>;
    }
    return (
        <div className={cn('overflow-hidden rounded-lg border border-white/[0.06]', className)}>
            <div
                className="hidden items-center gap-3 border-b border-white/[0.06] bg-white/[0.02] px-3 py-2 sm:grid"
                style={{ gridTemplateColumns: template }}
            >
                {columns.map((column) => (
                    <SubLabel className="mb-0" key={column.key}>
                        {column.label}
                    </SubLabel>
                ))}
                {arrow && <span />}
            </div>
            {rows.map((row, index) => (
                <Row
                    arrow={arrow}
                    columns={columns}
                    href={rowHref?.(row)}
                    key={rowKey(row)}
                    last={index === rows.length - 1}
                    row={row}
                    template={template}
                />
            ))}
        </div>
    );
}
