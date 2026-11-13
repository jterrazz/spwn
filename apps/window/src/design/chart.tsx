import type { Tone } from '../tone';
import type { Measured } from './measure';

import styles from './chart.module.css';

type Size = 'activity' | 'row' | 'tile';

export type ChartProps = {
    values: readonly (null | number)[];
    label: string;
    marks?: readonly string[] | undefined;
    size?: Size;
    tone?: Tone;
};

const FLOOR: Readonly<Record<Size, number>> = { activity: 3, row: 12, tile: 12 };

function heightOf(value: number, size: Size): Measured {
    return { '--height': `${String(Math.max(FLOOR[size], Math.min(100, value)))}%` };
}

export function Chart({ values, label, marks, size = 'tile', tone = 'plain' }: ChartProps) {
    return (
        <div
            aria-label={label}
            className={`${styles.chart} ${styles[size]} ${tone === 'plain' ? '' : styles[tone]}`}
            role="img"
        >
            {values.map((value, index) =>
                value === null ? (
                    <i
                        className={`${styles.mark} ${styles.gap}`}
                        key={index}
                        title={marks?.[index]}
                    />
                ) : (
                    <i
                        className={styles.mark}
                        key={index}
                        style={heightOf(value, size)}
                        title={marks?.[index]}
                    />
                ),
            )}
        </div>
    );
}
