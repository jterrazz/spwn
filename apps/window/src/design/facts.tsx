import type { ReactNode } from 'react';

import styles from './facts.module.css';

export type Fact = { label: string; value: ReactNode };

export function Facts({
    facts,
    layout = 'grid',
}: {
    facts: readonly Fact[];
    layout?: 'column' | 'grid' | 'line';
}) {
    return (
        <dl className={`${styles.facts} ${layout === 'grid' ? '' : styles[layout]}`}>
            {facts.map((fact) => (
                <div className={styles.fact} key={fact.label}>
                    <dt className={styles.label}>{fact.label}</dt>
                    <dd className={styles.value}>{fact.value}</dd>
                </div>
            ))}
        </dl>
    );
}
