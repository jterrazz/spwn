import type { ReactNode } from 'react';

import type { Identity } from './identity';
import type { MemoryHolder } from './memory-grid';

import styles from './memory-pick.module.css';

export type MemoryPickProps = {
    name: string;
    line: string;
    holders: readonly MemoryHolder[];
    chosen: number;
    identity?: Identity | undefined;
    free: number;
    children?: ReactNode;
};

function keysOf(prefix: string, squares: number): readonly string[] {
    const count = Number.isFinite(squares) ? Math.max(0, Math.round(squares)) : 0;
    return Array.from({ length: count }, (_, at) => `${prefix}-${String(at)}`);
}

export function MemoryPick({
    name,
    line,
    holders,
    chosen,
    identity,
    free,
    children,
}: MemoryPickProps) {
    return (
        <div className={styles.pick}>
            <div className={styles.head}>
                {chosen > 0 ? (
                    <b className={styles.size}>
                        {String(chosen)}
                        <small className={styles.unit}>GB</small>
                    </b>
                ) : null}
                {line === '' ? null : <span className={styles.line}>{line}</span>}
            </div>
            <div
                aria-label={`Memory of ${name}, one square per GB`}
                className={styles.grid}
                role="img"
            >
                {holders.map((holder) =>
                    keysOf(holder.key, holder.squares).map((key) => (
                        <i
                            className={`${styles.cell} ${holder.identity === 'system' ? '' : styles.held}`}
                            data-identity={holder.identity}
                            key={key}
                        />
                    )),
                )}
                {keysOf('chosen', chosen).map((key) => (
                    <i className={styles.cell} data-identity={identity} key={key} />
                ))}
                {keysOf('free', free).map((key) => (
                    <i className={`${styles.cell} ${styles.free}`} key={key} />
                ))}
            </div>
            {children}
        </div>
    );
}
