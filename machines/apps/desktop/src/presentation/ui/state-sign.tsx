import type { ReactNode } from 'react';

import type { Tone } from '../../domain/tone';

import styles from './state-sign.module.css';

const RING = { fill: 'none', stroke: 'currentColor' } as const;

// value: the landing's pictograms on a 16-unit box: calm a filled disc, warn a dot in a ring, new an arc being
// value: drawn, plain a hollow ring, bad a ring crossed. Each is drawn beside the word that says it.
const SIGNS: Record<Tone, ReactNode> = {
    bad: (
        <>
            <circle cx="8" cy="8" r="6.5" strokeWidth="1.2" {...RING} />
            <path
                d="M5.75 5.75 10.25 10.25M10.25 5.75 5.75 10.25"
                strokeLinecap="round"
                strokeWidth="1.5"
                {...RING}
            />
        </>
    ),
    new: (
        <>
            <circle cx="8" cy="8" opacity="0.3" r="5" strokeWidth="1.5" {...RING} />
            <circle
                cx="8"
                cy="8"
                r="5"
                strokeDasharray="15.7 31.4"
                strokeLinecap="round"
                strokeWidth="1.5"
                transform="rotate(-90 8 8)"
                {...RING}
            />
        </>
    ),
    ok: <circle cx="8" cy="8" fill="currentColor" r="4.5" />,
    plain: <circle cx="8" cy="8" r="4.5" strokeWidth="1.5" {...RING} />,
    warn: (
        <>
            <circle cx="8" cy="8" fill="currentColor" r="3" />
            <circle cx="8" cy="8" r="6.5" strokeWidth="1.2" {...RING} />
        </>
    ),
};

export type StateSignProps = { tone: Tone };

export function StateSign({ tone }: StateSignProps) {
    return (
        <svg
            aria-hidden="true"
            className={`${styles.sign} ${styles[tone]}`}
            height="14"
            viewBox="0 0 16 16"
            width="14"
        >
            {SIGNS[tone]}
        </svg>
    );
}
