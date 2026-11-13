import type { Tone } from '../tone';

import styles from './state-dot.module.css';

export function StateDot({ tone, word }: { tone: Tone; word: string }) {
    return (
        <span
            aria-label={word}
            className={`${styles.dot} ${styles[tone]}`}
            role="img"
            title={word}
        />
    );
}
