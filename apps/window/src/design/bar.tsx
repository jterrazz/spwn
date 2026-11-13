import type { Tone } from '../tone';
import type { Identity } from './identity';
import type { Measured } from './measure';

import styles from './bar.module.css';

export type BarProps = {
    percent: number;
    label: string;
    identity?: Identity | undefined;
    tone?: Tone;
    width?: 'fill' | 'fixed';
};

export function Bar({ percent, label, identity, tone = 'plain', width = 'fill' }: BarProps) {
    const shown = Math.max(2, Math.min(100, percent));
    const fill: Measured = { '--fill': `${String(shown)}%` };
    return (
        <div
            aria-label={label}
            aria-valuemax={100}
            aria-valuemin={0}
            aria-valuenow={Math.round(percent)}
            className={`${styles.bar} ${styles[tone]} ${width === 'fixed' ? styles.fixed : ''}`}
            data-identity={identity}
            role="meter"
        >
            <i className={styles.fill} style={fill} />
        </div>
    );
}

export type SharePart = {
    key: string;
    size: number;
    identity?: Identity | undefined;
    label?: string | undefined;
    rest?: boolean | undefined;
};

export type ShareBarProps = {
    parts: readonly SharePart[];
    label: string;
    of?: number | undefined;
};

function shareOf(part: SharePart, of: number | undefined): Measured {
    return of === undefined
        ? { '--size': part.size }
        : { '--share': `${((Math.max(0, part.size) / of) * 100).toFixed(2)}%` };
}

export function ShareBar({ parts, label, of }: ShareBarProps) {
    return (
        <div
            aria-label={label}
            className={`${styles.share} ${of === undefined ? '' : styles.whole}`}
            role="img"
        >
            {parts.map((part) => (
                <i
                    className={`${styles.part} ${part.rest === true ? styles.rest : ''}`}
                    data-identity={part.identity}
                    key={part.key}
                    style={shareOf(part, of)}
                    title={part.label}
                />
            ))}
        </div>
    );
}
