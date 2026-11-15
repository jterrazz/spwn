import type { ReactNode } from 'react';

import styles from './field.module.css';

export type FieldProps = {
    label: string;
    labelFor?: string | undefined;
    children: ReactNode;
    after?: ReactNode;
};

export function Field({ label, labelFor, children, after }: FieldProps) {
    const caption =
        labelFor === undefined ? (
            <span className={styles.label}>{label}</span>
        ) : (
            <label className={styles.label} htmlFor={labelFor}>
                {label}
            </label>
        );
    const inside = (
        <>
            <div className={styles.box}>
                {caption}
                {children}
            </div>
            {after}
        </>
    );
    return labelFor === undefined ? (
        <section aria-label={label} className={styles.field}>
            {inside}
        </section>
    ) : (
        <div className={styles.field}>{inside}</div>
    );
}
