import type { ReactNode } from 'react';

import styles from './missing.module.css';

export function Missing({ children }: { children: ReactNode }) {
    return (
        <p className={styles.missing}>
            <MissingMark />
            <span>{children}</span>
        </p>
    );
}

export function MissingMark({ children = 'Not given yet' }: { children?: string }) {
    return <span className={styles.mark}>{children}</span>;
}
