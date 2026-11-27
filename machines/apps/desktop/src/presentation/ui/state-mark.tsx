import { useLayoutEffect, useRef, useState } from 'react';

import type { Tone } from '../../domain/tone';
import { wordsThatFit } from './words-fit';

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

let ruler: CanvasRenderingContext2D | null | undefined;

function widthIn(element: HTMLElement): (text: string) => number {
    ruler ??= document.createElement('canvas').getContext('2d');
    const context = ruler;
    if (context === null) {
        return () => 0;
    }
    const style = getComputedStyle(element);
    // invariant: built from its longhands, since `font` reads empty once `font-variant-numeric` is set (document.css).
    context.font = `${style.fontStyle} ${style.fontWeight} ${style.fontSize} ${style.fontFamily}`;
    return (text) => context.measureText(text).width;
}

async function onceFacesLoad(fit: () => void): Promise<void> {
    try {
        await document.fonts.ready;
    } finally {
        fit();
    }
}

function FittedWord({ word, mark }: { word: string; mark: Mark }) {
    const ref = useRef<HTMLSpanElement>(null);
    const [shown, setShown] = useState(word);
    const last = useRef(word);
    useLayoutEffect(() => {
        const element = ref.current;
        if (element === null) {
            return;
        }
        const fit = () => {
            const next = wordsThatFit(
                word,
                element.getBoundingClientRect().width,
                widthIn(element),
            );
            if (next !== last.current) {
                last.current = next;
                setShown(next);
            }
        };
        fit();
        const observer = new ResizeObserver(fit);
        observer.observe(element);
        void onceFacesLoad(fit);
        return () => {
            observer.disconnect();
        };
    }, [word]);
    const cut = shown !== word;
    return (
        <>
            <span
                aria-hidden={cut ? true : undefined}
                className={`${styles.word} ${styles.fitted}`}
                data-mark={mark}
                ref={ref}
                title={cut ? word : undefined}
            >
                {shown}
            </span>
            {cut ? <span className={styles.unseen}>{word}</span> : null}
        </>
    );
}

// value: where the word is said: after the mark when unset, or `before` it, unless the state is calm; `after` the
// value: mark whatever the tone, for an outcome that reports the owner's own gesture; `name` only as the mark's
// value: name, where the place already says the state; `beside` by the sentence the caller draws next to it.
// value: `fit` keeps a word whole: `wrap` onto the lines it needs, or `words`, one line cut after its last whole
// value: word that fits, with `…`, the whole word its title and what a reader hears.
export type StateMarkProps =
    | { tone: Tone; said: 'beside' }
    | {
          tone: Tone;
          word: string;
          said?: 'after' | 'before' | 'name';
          fit?: 'words' | 'wrap' | undefined;
      };

export function StateMark(props: StateMarkProps) {
    const mark = MARKS[props.tone];
    if (props.said === 'beside') {
        return <span aria-hidden="true" className={styles.mark} data-mark={mark} />;
    }
    const { word, said, fit } = props;
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
        <span
            className={`${styles.state} ${said === undefined ? '' : styles[said]} ${fit === undefined ? '' : styles[fit]}`}
        >
            <span aria-hidden="true" className={styles.mark} data-mark={mark} />
            {fit === 'words' ? (
                <FittedWord mark={mark} word={word} />
            ) : (
                <span className={styles.word} data-mark={mark}>
                    {word}
                </span>
            )}
        </span>
    );
}
