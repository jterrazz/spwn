import type { ReactNode } from 'react';

import styles from './page.module.css';

export function Page({ children }: { children: ReactNode }) {
    return <div className={styles.page}>{children}</div>;
}

export function Tiles({ children }: { children: ReactNode }) {
    return <div className={styles.tiles}>{children}</div>;
}
