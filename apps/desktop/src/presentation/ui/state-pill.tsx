import type { Tone } from '../../domain/tone';

import styles from './state-pill.module.css';

const CALM: ReadonlySet<Tone> = new Set(['ok', 'plain']);

export function isCalm(tone: Tone): boolean {
    return CALM.has(tone);
}

export function StatePill({
    tone,
    word,
    quiet = false,
    dotted = true,
}: {
    tone: Tone;
    word: string;
    quiet?: boolean;
    dotted?: boolean;
}) {
    if (word.trim() === '') {
        throw new Error('A state pill carries its word: a colour alone is no state.');
    }
    if (quiet && CALM.has(tone)) {
        return <span className={styles.quiet}>{word}</span>;
    }
    return (
        <span className={`${styles.pill} ${styles[tone]} ${dotted ? '' : styles.undotted}`}>
            {word}
        </span>
    );
}
