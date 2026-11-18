import { useId } from 'react';

import styles from './radio-rows.module.css';

export type RadioRow<T extends string> = {
    id: T;
    title: string;
    line: string;
    held?: boolean;
};

export type RadioRowsProps<T extends string> = {
    label: string;
    rows: readonly RadioRow<T>[];
    value: null | T;
    onChange: (value: T) => void;
};

export function RadioRows<T extends string>({ label, rows, value, onChange }: RadioRowsProps<T>) {
    const id = useId();
    return (
        <div aria-label={label} className={styles.rows} role="radiogroup">
            {rows.map((row) => {
                const held = row.held === true;
                const titled = `${id}-${row.id}-title`;
                const said = `${id}-${row.id}-line`;
                return (
                    <label
                        className={`${styles.row} ${held ? styles.held : styles.open}`}
                        key={row.id}
                    >
                        <input
                            aria-describedby={row.line === '' ? undefined : said}
                            aria-labelledby={titled}
                            checked={row.id === value}
                            className={styles.ring}
                            disabled={held}
                            name={id}
                            onChange={() => {
                                onChange(row.id);
                            }}
                            type="radio"
                            value={row.id}
                        />
                        <span className={styles.title} id={titled}>
                            {row.title}
                        </span>
                        {row.line === '' ? null : (
                            <span className={styles.line} id={said}>
                                {row.line}
                            </span>
                        )}
                    </label>
                );
            })}
        </div>
    );
}
