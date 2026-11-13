import type { ReactNode } from 'react';

import { Caption } from '../../design/index';

import styles from './gallery.module.css';

export const nothing = () => undefined;

export function Section({ title, children }: { title: string; children: ReactNode }) {
    return (
        <section aria-label={title} className={styles.section}>
            <Caption rule>{title}</Caption>
            <div className={styles.samples}>{children}</div>
        </section>
    );
}
