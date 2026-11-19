import type { ReactNode } from 'react';

import styles from './panel.module.css';

export type PanelProps = {
    title: string;
    note?: string;
    action?: ReactNode;
    padded?: boolean;
    children?: ReactNode;
};

export function Panel({ title, note, action, padded = false, children }: PanelProps) {
    return (
        <section aria-label={title} className={`${styles.panel} ${padded ? styles.padded : ''}`}>
            <div className={styles.head}>
                <h2 className={styles.title}>
                    {title}
                    {(note ?? '') === '' ? null : <span className={styles.note}>{note}</span>}
                </h2>
                {action}
            </div>
            {children ?? <p className={styles.empty}>Nothing here yet.</p>}
        </section>
    );
}
