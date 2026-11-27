import { memo, useLayoutEffect, useRef } from 'react';
import type { ReactNode } from 'react';

import { SEPARATOR } from '../../../domain/separator';
import { useMotion } from '../canvas/motion';
import { Icon } from '../icon';

import styles from './strip.module.css';

// value: what the strip says around the page's own words: its accessible name for a title, the Open button, the
// value: hint before it, and a count of needs.
export type StripWords = {
    open: string;
    hint: string;
    name: (title: string) => string;
    needs: (count: number) => string;
};

export type StripProps = {
    title: string;
    line?: string | undefined;
    meta: readonly string[];
    needs: number;
    icon?: ReactNode;
    words: StripWords;
    onOpen: () => void;
};

// value: the prototype's strip text arriving (`drawStrip`): 280 ms, 6 px up, on --ease.
const ARRIVE = { duration: 280, rise: 6 };

function easeOf(element: Element): string {
    const value = getComputedStyle(element).getPropertyValue('--ease').trim();
    return value === '' ? 'linear' : value;
}

// invariant: the parts of the meta row are joined by the kit's one separator, drawn as its own span so the row keeps
// invariant: the prototype's 8 px around it.
function Meta({ icon, meta, needs, words }: Pick<StripProps, 'icon' | 'meta' | 'needs' | 'words'>) {
    const mark = SEPARATOR.trim();
    return (
        <span className={styles.meta}>
            {icon === undefined ? null : <span className={styles.icon}>{icon}</span>}
            {meta.map((part, at) => (
                <span className={styles.part} key={part}>
                    {at > 0 ? <span aria-hidden="true">{mark}</span> : null}
                    <span>{part}</span>
                </span>
            ))}
            {needs > 0 ? (
                <span className={styles.part}>
                    {meta.length > 0 ? <span aria-hidden="true">{mark}</span> : null}
                    <span className={styles.needs}>
                        <i className={styles.mark} />
                        {words.needs(needs)}
                    </span>
                </span>
            ) : null}
        </span>
    );
}

// value: the page as it peeks at the foot of the space: its meta row, its title and, when given, its line, with
// value: Open at its right. It is one button named "Open <title>"; a new page's text arrives as the prototype's does.
function StripOf({ icon, line, meta, needs, onOpen, title, words }: StripProps) {
    const motion = useMotion();
    const text = useRef<HTMLSpanElement | null>(null);
    const shown = useRef<null | string>(null);
    const key = `${title}\n${line ?? ''}`;
    useLayoutEffect(() => {
        const before = shown.current;
        shown.current = key;
        const element = text.current;
        if (before === null || before === key || !motion || element === null) {
            return;
        }
        element.animate(
            [
                { opacity: 0, transform: `translateY(${String(ARRIVE.rise)}px)` },
                { opacity: 1, transform: 'none' },
            ],
            { duration: ARRIVE.duration, easing: easeOf(element) },
        );
    }, [key, motion]);

    return (
        <button
            aria-label={words.name(title)}
            className={styles.strip}
            data-peek={line === undefined ? 'title' : 'line'}
            onClick={onOpen}
            type="button"
        >
            <span className={styles.text} ref={text}>
                <Meta icon={icon} meta={meta} needs={needs} words={words} />
                <span className={styles.title}>{title}</span>
                {line === undefined ? null : <span className={styles.line}>{line}</span>}
            </span>
            <span className={styles.go}>
                <span className={styles.hint}>{words.hint}</span>
                <span className={styles.open}>
                    <Icon name="up" />
                    {words.open}
                </span>
            </span>
        </button>
    );
}

export const Strip = memo(StripOf);
