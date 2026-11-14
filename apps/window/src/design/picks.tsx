import { useId } from 'react';

import styles from './picks.module.css';

export type Pick<T extends string> = {
    id: T;
    title: string;
    lines?: readonly string[];
    disabled?: boolean;
};

export type PicksProps<T extends string> = {
    label: string;
    look: 'cards' | 'chips';
    options: readonly Pick<T>[];
    value: null | T;
    onChange: (value: T) => void;
};

export function Picks<T extends string>({ label, look, options, value, onChange }: PicksProps<T>) {
    const id = useId();
    return (
        <div aria-label={label} className={styles[look]} role="group">
            {options.map((option) => {
                const lines = option.lines ?? [];
                const described = `${id}-${option.id}-lines`;
                const titled = `${id}-${option.id}-title`;
                return (
                    <button
                        aria-describedby={lines.length > 0 ? described : undefined}
                        aria-labelledby={titled}
                        aria-pressed={option.id === value}
                        className={styles.pick}
                        disabled={option.disabled === true}
                        key={option.id}
                        onClick={() => {
                            onChange(option.id);
                        }}
                        type="button"
                    >
                        <b className={styles.title} id={titled}>
                            {option.title}
                        </b>
                        {lines.length > 0 ? (
                            <span className={styles.lines} id={described}>
                                {lines.map((line) => (
                                    <span key={line}>{line}</span>
                                ))}
                            </span>
                        ) : null}
                    </button>
                );
            })}
        </div>
    );
}
