import type { Identity } from './identity';

import styles from './switch.module.css';

export type SwitchOption<T extends string> = {
    id: T;
    label: string;
    count?: number | undefined;
    identity?: Identity | undefined;
};

type Shared<T extends string> = {
    label: string;
    options: readonly SwitchOption<T>[];
    variant?: 'filter' | 'tabs';
    surface?: 'bare' | 'canvas' | 'panel';
    disabled?: boolean;
    wrap?: boolean;
    controls?: string;
};

export type SwitchProps<T extends string> = Shared<T> &
    (
        | { all: string; value: null | T; onChange: (value: null | T) => void }
        | { all?: undefined; value: T; onChange: (value: T) => void }
    );

function Option({
    label,
    count,
    identity,
    chosen,
    tabs,
    disabled,
    controls,
    onPick,
}: {
    label: string;
    count?: number | undefined;
    identity?: Identity | undefined;
    chosen: boolean;
    tabs: boolean;
    disabled: boolean;
    controls: string | undefined;
    onPick: () => void;
}) {
    return (
        <button
            aria-controls={tabs ? controls : undefined}
            aria-pressed={tabs ? undefined : chosen}
            aria-selected={tabs ? chosen : undefined}
            className={styles.option}
            disabled={disabled}
            onClick={onPick}
            role={tabs ? 'tab' : undefined}
            tabIndex={tabs && !chosen ? -1 : undefined}
            type="button"
        >
            {identity === undefined ? null : (
                <span aria-hidden="true" className={styles.dot} data-identity={identity} />
            )}
            {label}
            {count === undefined ? null : <span className={styles.count}>{count}</span>}
        </button>
    );
}

export function Switch<T extends string>(props: SwitchProps<T>) {
    const {
        label,
        options,
        variant = 'filter',
        surface = 'canvas',
        disabled = false,
        wrap = false,
        controls,
    } = props;
    const tabs = variant === 'tabs';
    return (
        <div
            aria-disabled={disabled ? true : undefined}
            aria-label={label}
            className={[
                styles.switch,
                styles[variant],
                surface === 'canvas' ? '' : styles[surface === 'panel' ? 'inPanel' : 'bare'],
                wrap ? styles.wrap : '',
            ].join(' ')}
            role={tabs ? 'tablist' : 'group'}
        >
            {props.all === undefined ? null : (
                <Option
                    chosen={props.value === null}
                    controls={controls}
                    disabled={disabled}
                    label={props.all}
                    onPick={() => {
                        props.onChange(null);
                    }}
                    tabs={tabs}
                />
            )}
            {options.map((option) => (
                <Option
                    chosen={option.id === props.value}
                    controls={controls}
                    count={option.count}
                    disabled={disabled}
                    identity={option.identity}
                    key={option.id}
                    label={option.label}
                    onPick={() => {
                        props.onChange(option.id);
                    }}
                    tabs={tabs}
                />
            ))}
        </div>
    );
}
