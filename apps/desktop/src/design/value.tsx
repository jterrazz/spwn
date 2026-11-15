import type { Tone } from '../tone';
import type { Identity } from './identity';
import { shownOrWithheld } from './secret';

import styles from './value.module.css';

const KEPT_END = 18;

export type ValueProps = {
    children: string;
    look?: 'boxed' | 'plain' | 'raised';
    identity?: Identity | undefined;
    tone?: Tone | undefined;
};

export function Value({ children, look = 'plain', identity, tone }: ValueProps) {
    const shown = shownOrWithheld(children);
    const cut = Math.max(0, shown.length - Math.min(KEPT_END, Math.floor(shown.length / 2)));
    return (
        <span
            className={`${styles.value} ${look === 'plain' ? '' : styles[look]} ${tone === undefined ? '' : styles.toned}`}
            data-identity={identity}
            data-tone={tone}
            title={shown}
        >
            {identity === undefined ? null : <span aria-hidden="true" className={styles.square} />}
            <span className={styles.whole}>{shown}</span>
            <span aria-hidden className={styles.start}>
                {shown.slice(0, cut)}
            </span>
            <span aria-hidden className={styles.finish}>
                {shown.slice(cut)}
            </span>
        </span>
    );
}
