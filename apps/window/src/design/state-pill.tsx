import type { Tone } from '../tone';
import { capitalised } from './case';

import styles from './state-pill.module.css';

const CALM: ReadonlySet<Tone> = new Set(['ok', 'plain']);

export function StatePill({
    tone,
    word,
    quiet = false,
}: {
    tone: Tone;
    word: string;
    quiet?: boolean;
}) {
    if (word.trim() === '') {
        throw new Error('A state pill carries its word: a colour alone is no state.');
    }
    if (quiet && CALM.has(tone)) {
        return <span className={styles.quiet}>{capitalised(word)}</span>;
    }
    return <span className={`${styles.pill} ${styles[tone]}`}>{capitalised(word)}</span>;
}
