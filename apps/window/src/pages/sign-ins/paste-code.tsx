import type { AccountsStep } from '@jterrazz/station-api';
import { useEffect, useRef, useState } from 'react';
import type { RefObject } from 'react';

import { Button } from '../../design/index';
import { EngineError } from '../../engine/engine-error';
import { useClient } from '../../engine/react';
import type { Found } from '../../model/machines';
import { Unanswered } from '../kit/readings';
import { aimed } from './aim';

import styles from './sign-ins.module.css';

function refusalOf(error: unknown): EngineError {
    return error instanceof EngineError ? error : new EngineError(undefined, String(error));
}

type Answering = {
    input: RefObject<HTMLInputElement | null>;
    sending: boolean;
    refused: EngineError | undefined;
    send: () => Promise<void>;
};

// invariant: the pasted code lives in the input alone: it is read from the element in the one frame that sends
// invariant: it, the input is emptied before the call leaves, and no state, log or recording ever holds it.
function useAnswer(found: Found, step: AccountsStep): Answering {
    const client = useClient();
    const input = useRef<HTMLInputElement>(null);
    const [sending, setSending] = useState(false);
    const [refused, setRefused] = useState<EngineError | undefined>(undefined);
    useEffect(() => {
        const element = input.current;
        return () => {
            if (element) {
                element.value = '';
            }
        };
    }, []);
    const send = async () => {
        const element = input.current;
        if (element === null || element.value === '') {
            return;
        }
        const { input: asked, target } = aimed(found, {
            Code: element.value,
            Exchange: step.Exchange,
        });
        element.value = '';
        setSending(true);
        setRefused(undefined);
        try {
            await client.call('accounts.answer', asked, { target });
        } catch (error) {
            setRefused(refusalOf(error));
        }
        setSending(false);
    };
    return { input, refused, send, sending };
}

export function PasteCode({ found, step }: { found: Found; step: AccountsStep }) {
    const { input, refused, send, sending } = useAnswer(found, step);
    return (
        <form
            className={styles.paste}
            onSubmit={(event) => {
                event.preventDefault();
                void send();
            }}
        >
            <input
                aria-label={`Code from ${step.Site ?? ''}`}
                autoComplete="off"
                className={styles.input}
                ref={input}
                spellCheck={false}
                type="password"
            />
            {refused === undefined ? null : <Unanswered error={refused} size="panel" />}
            <span className={styles.label}>
                <Button
                    busy={sending}
                    onClick={() => {
                        void send();
                    }}
                    variant="primary"
                >
                    Continue
                </Button>
            </span>
        </form>
    );
}
