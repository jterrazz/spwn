import styles from './tag.module.css';

export function TypeTag({ children }: { children: string }) {
    return <span className={styles.type}>{children}</span>;
}
