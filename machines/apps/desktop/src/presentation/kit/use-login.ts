import type { AccountsProgramRow, AccountsStep } from '@jterrazz/station-api';
import { useEffect, useRef, useState } from 'react';

import type { EngineError } from '../../domain/engine-error';
import { aimed } from '../../domain/machines/aim';
import type { Found } from '../../domain/machines/machines';
import { useClient } from '../dependencies/dependencies';

export type Seen =
    | { kind: 'asking' }
    | { kind: 'failed'; error: EngineError; step: AccountsStep | undefined }
    | { kind: 'step'; step: AccountsStep };

// invariant: one sign-in is one watch of accounts.login, opened when its sheet is drawn and closed when it goes: a
// invariant: closed watch sends its cancel, so the engine drops the exchange with the sheet. A watch that ends in
// invariant: an error is closed at once, never opened again behind the sheet's back. A new code is a new sheet.
export function useLogin(
    found: Found,
    program: AccountsProgramRow['Program'],
    onSignedIn: () => void,
): Seen {
    const client = useClient();
    const [seen, setSeen] = useState<Seen>({ kind: 'asking' });
    const latest = useRef({ found, onSignedIn });
    useEffect(() => {
        latest.current = { found, onSignedIn };
    });
    useEffect(() => {
        const { input, target } = aimed(latest.current.found, {
            Program: program,
        });
        let ended = false;
        let last: AccountsStep | undefined;
        const held: { close?: () => void } = {};
        const end = () => {
            ended = true;
            held.close?.();
        };
        const watch = client.watch(
            'accounts.login',
            input,
            {
                error: (error) => {
                    end();
                    setSeen({ error, kind: 'failed', step: last });
                },
                out: (out) => {
                    if (out.Step !== undefined) {
                        last = out.Step;
                        setSeen({ kind: 'step', step: out.Step });
                    } else if (out.Programs !== undefined) {
                        end();
                        latest.current.onSignedIn();
                    }
                },
            },
            target,
        );
        held.close = () => {
            watch.close();
        };
        if (ended) {
            watch.close();
        }
        return () => {
            watch.close();
        };
    }, [client, program]);
    return seen;
}
