import styles from './field.module.css';

export function Field({ children }: { children: string }) {
    return <span className={styles.field}>{children}</span>;
}
