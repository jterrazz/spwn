import { createContext, useContext, useId, useLayoutEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';

import styles from './button.module.css';

// value: the six looks of a button, named once: primary leads a screen, plain is any other gesture, link reads
// value: as a sentence, inline is a link inside a grey line of small text, danger opens a destruction, and
// value: destroy, the filled danger, only confirms one.
export const BUTTON_VARIANTS = ['primary', 'plain', 'link', 'inline', 'danger', 'destroy'] as const;

export type ButtonVariant = (typeof BUTTON_VARIANTS)[number];

type Refuse = (refused: boolean) => void;

type Scope = { claim: (id: string, refused: Refuse, yields: boolean) => () => void };

const SECOND_PRIMARY = 'One primary button per screen: a second one was drawn.';

const PrimaryScope = createContext<null | Scope>(null);

// invariant: a primary that yields (a machine header's) is drawn plain while a page's own primary holds the
// invariant: screen, and primary again once it goes.
function scopeOfOnePrimary(strict: boolean): Scope {
    const holders = new Set<string>();
    const yielding = new Map<string, Refuse>();
    const yieldAll = (refused: boolean) => {
        for (const refuse of yielding.values()) {
            refuse(refused);
        }
    };
    return {
        claim: (id, refused, yields) => {
            if (yields) {
                yielding.set(id, refused);
                refused(holders.size > 0);
                return () => {
                    yielding.delete(id);
                };
            }
            holders.add(id);
            if (holders.size > 1) {
                holders.delete(id);
                if (strict) {
                    throw new Error(SECOND_PRIMARY);
                }
                globalThis.reportError(new Error(`${SECOND_PRIMARY} It is drawn plain.`));
                refused(true);
                return () => undefined;
            }
            yieldAll(true);
            return () => {
                holders.delete(id);
                yieldAll(holders.size > 0);
            };
        },
    };
}

const STRICT = import.meta.env.MODE !== 'production';

export function Screen({ children, strict = STRICT }: { children: ReactNode; strict?: boolean }) {
    const scope = useMemo(() => scopeOfOnePrimary(strict), [strict]);
    return <PrimaryScope value={scope}>{children}</PrimaryScope>;
}

export type ButtonProps = {
    children: ReactNode;
    variant?: ButtonVariant;
    onClick?: (() => void) | undefined;
    disabled?: boolean;
    busy?: boolean;
    name?: string | undefined;
    title?: string | undefined;
    yields?: boolean;
};

export function Button({
    children,
    variant = 'plain',
    onClick,
    disabled = false,
    busy = false,
    name,
    title,
    yields = false,
}: ButtonProps) {
    const scope = useContext(PrimaryScope);
    const id = useId();
    const [refused, setRefused] = useState(false);
    const primary = variant === 'primary';
    useLayoutEffect(
        () => (primary && scope ? scope.claim(id, setRefused, yields) : undefined),
        [primary, scope, id, yields],
    );
    const drawn = primary && refused ? 'plain' : variant;
    return (
        <button
            aria-busy={busy || undefined}
            aria-label={name}
            className={`${styles.button} ${styles[drawn]} ${busy ? styles.busy : ''}`}
            disabled={disabled || busy}
            onClick={onClick}
            title={title}
            type="button"
        >
            {busy ? <i aria-hidden="true" className={styles.spinner} /> : null}
            {children}
        </button>
    );
}
