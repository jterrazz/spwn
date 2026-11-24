import { useId } from 'react';
import type { ReactNode } from 'react';

import { MachineIcon } from './machine-icon';

import styles from './mac-section.module.css';

export type MacSectionProps = { name: string; children: ReactNode };

// value: one Mac's part of a page that shows every Mac together, headed by its name.
export function MacSection({ name, children }: MacSectionProps) {
    const id = useId();
    return (
        <section aria-labelledby={id} className={styles.section}>
            <h2 className={styles.name}>
                <MachineIcon kind="mac" />
                <span id={id}>{name}</span>
            </h2>
            {children}
        </section>
    );
}
