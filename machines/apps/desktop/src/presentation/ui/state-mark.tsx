import type { Tone } from '../../domain/tone';

import styles from './state-mark.module.css';

type Mark = 'bad' | 'busy' | 'off' | 'ok' | 'you';

// value: each tone's mark, a shape of its own so a state reads in greys: ok a filled disc, warn the diamond that is
// value: yours, bad a square, new a half disc that pulses, plain a ring.
const MARKS: Readonly<Record<Tone, Mark>> = {
    bad: 'bad',
    new: 'busy',
    ok: 'ok',
    plain: 'off',
    warn: 'you',
};

// invariant: the one list of calm tones: a calm state draws no word, its mark carries the word as its name.
const CALM: ReadonlySet<Tone> = new Set(['ok']);

export function isCalm(tone: Tone): boolean {
    return CALM.has(tone);
}

// value: where the word is said: after the mark when unset, or `before` it, unless the state is calm; `after` the
// value: mark whatever the tone, for an outcome that reports the owner's own gesture; `name` only as the mark's
// value: name, where the place already says the state; `beside` by the sentence the caller draws next to it.
export type StateMarkProps =
    | { tone: Tone; said: 'beside' }
    | { tone: Tone; word: string; said?: 'after' | 'before' | 'name' };

export function StateMark(props: StateMarkProps) {
    const mark = MARKS[props.tone];
    if (props.said === 'beside') {
        return <span aria-hidden="true" className={styles.mark} data-mark={mark} />;
    }
    const { word, said } = props;
    if (word.trim() === '') {
        throw new Error('A state mark carries its word: a colour alone is no state.');
    }
    if (said === 'name' || (said !== 'after' && isCalm(props.tone))) {
        return (
            <span
                aria-label={word}
                className={styles.mark}
                data-mark={mark}
                role="img"
                title={word}
            />
        );
    }
    return (
        <span className={`${styles.state} ${said === undefined ? '' : styles[said]}`}>
            <span aria-hidden="true" className={styles.mark} data-mark={mark} />
            <span className={styles.word} data-mark={mark}>
                {word}
            </span>
        </span>
    );
}
