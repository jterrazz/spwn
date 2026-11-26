import styles from './guide.module.css';

export function HideMark() {
    return (
        <svg aria-hidden="true" className={styles.hideMark} viewBox="0 0 12 12">
            <path d="M3 3l6 6M9 4.5V9H4.5" />
        </svg>
    );
}

export function Fingerprint() {
    return (
        <svg aria-hidden="true" className={styles.fingerprint} viewBox="0 0 24 24">
            <path d="M6.5 8.5a6.5 6.5 0 0 1 11 0" />
            <path d="M5 13c0-3.9 3.1-7 7-7s7 3.1 7 7c0 1.5-.3 2.9-.8 4.2" />
            <path d="M8.2 18.5A7 7 0 0 1 8 13a4 4 0 0 1 8 0c0 2-.4 3.8-1.1 5.4" />
            <path d="M12 13c0 2.6-.5 5-1.5 7" />
        </svg>
    );
}

export function KeptMark() {
    return (
        <svg aria-hidden="true" className={styles.keptMark} viewBox="0 0 16 16">
            <path d="M8 2l5.5 3v6L8 14l-5.5-3V5z" />
            <path d="M2.5 5L8 8l5.5-3M8 8v6" />
        </svg>
    );
}
