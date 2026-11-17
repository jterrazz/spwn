import type { ReactNode } from 'react';

import { SEPARATOR } from '../../domain/separator';
import type { Tone } from '../../domain/tone';

import styles from './tile.module.css';

export type TileProps = {
    label: string;
    value: ReactNode;
    unit?: string;
    foot: ReactNode;
    footWrapped?: { title: string };
    footState?: undefined | { tone: Tone; word: string };
    viz?: ReactNode;
    aside?: { text: string; tone: 'ok' | 'plain' };
};

export function Tile({ label, value, unit, foot, footWrapped, footState, viz, aside }: TileProps) {
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
            <div
                className={
                    footWrapped === undefined ? styles.foot : `${styles.foot} ${styles.wrapped}`
                }
                title={footWrapped?.title}
            >
                {footState === undefined ? null : (
                    <span className={`${styles.footState} ${styles[footState.tone]}`}>
                        {footState.word}
                    </span>
                )}
                {footState === undefined || foot === '' ? null : SEPARATOR}
                {foot}
            </div>
        </div>
    );
}
