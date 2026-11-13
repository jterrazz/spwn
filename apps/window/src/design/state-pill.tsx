import type { Tone } from '../tone';
import { capitalised } from './case';

import styles from './state-pill.module.css';

const CALM: ReadonlySet<Tone> = new Set(['ok', 'plain']);

export function isCalm(tone: Tone): boolean {
    return CALM.has(tone);
}

export function StatePill({
    tone,
    word,
    quiet = false,
    asWritten = false,
    dotted = true,
}: {
    tone: Tone;
    word: string;
    quiet?: boolean;
    asWritten?: boolean;
    dotted?: boolean;
}) {
    if (word.trim() === '') {
        throw new Error('A state pill carries its word: a colour alone is no state.');
    }
    const shown = asWritten ? word : capitalised(word);
    if (quiet && CALM.has(tone)) {
        return <span className={styles.quiet}>{shown}</span>;
    }
    return (
        <span className={`${styles.pill} ${styles[tone]} ${dotted ? '' : styles.undotted}`}>
            {shown}
        </span>
    );
}
