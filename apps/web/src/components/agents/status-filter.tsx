'use client';

import type { StatusFilter } from './roster';

const FILTERS: StatusFilter[] = ['all', 'deployed', 'limbo'];

/** All, deployed or in limbo, each with its count. */
export function StatusFilters({
    filter,
    counts,
    onFilter,
}: {
    filter: StatusFilter;
    counts: Record<StatusFilter, number>;
    onFilter: (filter: StatusFilter) => void;
}) {
    return (
        <div className="flex flex-wrap items-center gap-3">
            <div className="glass-pill flex items-center gap-1 px-1 py-1">
                {FILTERS.map((entry) => (
                    <button
                        className={`rounded-full px-3 py-1 text-xs capitalize transition-colors ${filter === entry ? 'text-foreground/90 bg-white/[0.1]' : 'text-muted-foreground/50 hover:text-foreground/70'}`}
                        key={entry}
                        onClick={() => {
                            onFilter(entry);
                        }}
                        type="button"
                    >
                        {entry}{' '}
                        <span className="text-muted-foreground/40 ml-1 font-mono text-[10px]">
                            {counts[entry]}
                        </span>
                    </button>
                ))}
            </div>
        </div>
    );
}
