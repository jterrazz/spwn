import { useState } from 'react';

import type { ButtonVariant } from '../../design/button';
import { Button } from '../../design/index';
import { looksSecret } from '../../design/secret';
import type { EngineError } from '../../engine/engine-error';
import type { Openable } from '../../engine/opener';
import { useOpener } from '../../engine/react';
import { Unanswered } from './readings';

import styles from '../page.module.css';

export const WITHHELD_OPENS = 'Not opened: what it would open looks like a secret';

export type OpenButtonProps = {
    label: string;
    openable: Openable | undefined;
    held: string;
    variant?: ButtonVariant;
    yields?: boolean;
};

// invariant: a click opens the engine's value and marks nothing done: the window changed nothing, and the reads
// invariant: show what the engine says once it does. A value shaped like a secret is never opened, as never copied.
export function OpenButton({
    label,
    openable,
    held,
    variant = 'plain',
    yields = false,
}: OpenButtonProps) {
    const opener = useOpener();
    const [busy, setBusy] = useState(false);
    const [refused, setRefused] = useState<EngineError | undefined>(undefined);
    const withheld = openable !== undefined && looksSecret(openable.value);
    if (openable === undefined || withheld) {
        return (
            <Button
                disabled
                title={withheld ? WITHHELD_OPENS : held}
                variant={variant}
                yields={yields}
            >
                {label}
            </Button>
        );
    }
    const open = async () => {
        setBusy(true);
        setRefused(undefined);
        const opened = await opener(openable);
        setBusy(false);
        setRefused(opened.kind === 'refused' ? opened.error : undefined);
    };
    return (
        <>
            <Button
                busy={busy}
                onClick={() => {
                    void open();
                }}
                title={openable.value}
                variant={variant}
                yields={yields}
            >
                {label}
            </Button>
            {refused === undefined ? null : (
                <span className={styles.outcome}>
                    <Unanswered error={refused} size="panel" />
                    <Button
                        onClick={() => {
                            setRefused(undefined);
                        }}
                        variant="link"
                    >
                        Dismiss
                    </Button>
                </span>
            )}
        </>
    );
}
