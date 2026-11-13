import styles from './caption.module.css';

export function Caption({ children, rule = false }: { children: string; rule?: boolean }) {
    return <p className={`${styles.caption} ${rule ? styles.rule : ''}`}>{children}</p>;
}
