import type { Identity } from './identity';

import styles from './switch.module.css';

export type SwitchOption<T extends string> = {
    id: T;
    label: string;
    count?: number | undefined;
    identity?: Identity | undefined;
    flagged?: string | undefined;
    busy?: boolean | undefined;
};

type Shared<T extends string> = {
    label: string;
    options: readonly SwitchOption<T>[];
    variant?: 'filter' | 'tabs';
    surface?: 'bare' | 'canvas' | 'panel';
    disabled?: boolean;
    wrap?: boolean;
    even?: boolean;
    controls?: string;
};

export type SwitchProps<T extends string> = Shared<T> &
    (
        | { all: string; value: null | T; onChange: (value: null | T) => void }
        | { all?: undefined; value: null | T; onChange: (value: T) => void }
    );

type Face = {
    label: string;
    count?: number | undefined;
    identity?: Identity | undefined;
    flagged?: string | undefined;
    busy?: boolean | undefined;
};

function OptionFace({ label, count, identity, flagged, busy }: Face) {
    return (
        <>
            {busy === true ? <i aria-hidden="true" className={styles.spinner} /> : null}
            {identity === undefined ? null : (
                <span aria-hidden="true" className={styles.dot} data-identity={identity} />
            )}
            {label}
            {count === undefined ? null : <span className={styles.count}>{count}</span>}
            {flagged === undefined ? null : (
                <span aria-label={flagged} className={styles.flag} role="img" />
            )}
        </>
    );
}

function Option({
    chosen,
    tabs,
    disabled,
    controls,
    onPick,
    ...face
}: Face & {
    chosen: boolean;
    tabs: boolean;
    disabled: boolean;
    controls: string | undefined;
    onPick: () => void;
}) {
    const busy = face.busy === true;
    return (
        <button
            aria-busy={busy || undefined}
            aria-controls={tabs ? controls : undefined}
            aria-pressed={tabs ? undefined : chosen}
            aria-selected={tabs ? chosen : undefined}
            className={styles.option}
            disabled={disabled || busy}
            onClick={onPick}
            role={tabs ? 'tab' : undefined}
            tabIndex={tabs && !chosen ? -1 : undefined}
            type="button"
        >
            <OptionFace {...face} />
        </button>
    );
}

function lookOf<T extends string>({
    variant = 'filter',
    surface = 'canvas',
    wrap = false,
    even = false,
}: Shared<T>): string {
    return [
        styles.switch,
        styles[variant],
        surface === 'canvas' ? '' : styles[surface === 'panel' ? 'inPanel' : 'bare'],
        wrap ? styles.wrap : '',
        even ? styles.even : '',
    ].join(' ');
}

export function Switch<T extends string>(props: SwitchProps<T>) {
    const { label, options, variant = 'filter', disabled = false, controls } = props;
    const tabs = variant === 'tabs';
    return (
        <div
            aria-disabled={disabled ? true : undefined}
            aria-label={label}
            className={lookOf(props)}
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
                    busy={option.busy}
                    chosen={option.id === props.value}
                    controls={controls}
                    count={option.count}
                    disabled={disabled}
                    flagged={option.flagged}
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
