import type { ReactNode } from 'react';

import type { Tone } from '../tone';

import styles from './tile.module.css';

export type TileProps = {
    label: string;
    value: ReactNode;
    unit?: string;
    foot: ReactNode;
    footState?: undefined | { tone: Tone; word: string };
    viz?: ReactNode;
    aside?: { text: string; tone: 'ok' | 'plain' };
};

const JOINED = ' · ';

export function Tile({ label, value, unit, foot, footState, viz, aside }: TileProps) {
    return (
        <div className={styles.tile}>
            <div className={styles.label}>
                <span>{label}</span>
                {aside ? (
                    <span className={aside.tone === 'ok' ? styles.asideOk : undefined}>
                        {aside.text}
                    </span>
                ) : null}
            </div>
            <div className={styles.mid}>
                <div className={styles.number}>
                    {value}
                    {(unit ?? '') === '' ? null : <small className={styles.unit}>{unit}</small>}
                </div>
                {viz}
            </div>
            <div className={styles.foot}>
                {footState === undefined ? null : (
                    <span className={`${styles.footState} ${styles[footState.tone]}`}>
                        {footState.word}
                    </span>
                )}
                {footState === undefined || foot === '' ? null : JOINED}
                {foot}
            </div>
        </div>
    );
}
