import type { Identity } from './identity';

import styles from './memory-grid.module.css';

export type MemoryHolder = {
    key: string;
    label: string;
    squares: number;
    identity?: Identity | undefined;
    pending?: boolean | undefined;
};

export type MemoryGridProps = {
    name: string;
    total: number;
    free: number;
    holders: readonly MemoryHolder[];
};

function count(squares: number): number {
    return Number.isFinite(squares) ? Math.max(0, Math.round(squares)) : 0;
}

function cellsOf(holder: MemoryHolder): readonly string[] {
    return Array.from({ length: count(holder.squares) }, (_, at) => `${holder.key}-${String(at)}`);
}

export function MemoryGrid({ name, total, free, holders }: MemoryGridProps) {
    const freeCells = Array.from({ length: count(free) }, (_, at) => `free-${String(at)}`);
    return (
        <div className={styles.memory}>
            <div className={styles.head}>
                <b>{name}</b>
                <span>
                    <em>{`${String(count(free))} GB free`}</em>
                    {` of ${String(count(total))}`}
                </span>
            </div>
            <div
                aria-label={`Memory of ${name}, one square per GB`}
                className={styles.grid}
                role="img"
            >
                {holders.map((holder) =>
                    cellsOf(holder).map((key) => (
                        <i
                            className={`${styles.cell} ${holder.pending === true ? styles.pending : ''}`}
                            data-identity={holder.identity}
                            key={key}
                        />
                    )),
                )}
                {freeCells.map((key) => (
                    <i className={`${styles.cell} ${styles.free}`} key={key} />
                ))}
            </div>
            <div className={styles.legend}>
                {holders.map((holder) => (
                    <span data-identity={holder.identity} key={holder.key}>
                        <b>{String(count(holder.squares))}</b> {holder.label}
                    </span>
                ))}
                <span className={styles.free}>
                    <b>{String(count(free))}</b> free
                </span>
            </div>
        </div>
    );
}
