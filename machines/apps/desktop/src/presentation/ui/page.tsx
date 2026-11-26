import type { ReactNode } from 'react';

import styles from './page.module.css';

export type Density = 'comfortable' | 'compact';

export function Page({ children, density }: { children: ReactNode; density: Density }) {
    return (
        <div className={styles.page} data-density={density}>
            {children}
        </div>
    );
}

export function Tiles({ children }: { children: ReactNode }) {
    return <div className={styles.tiles}>{children}</div>;
}
