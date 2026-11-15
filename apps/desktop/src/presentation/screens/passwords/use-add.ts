import type { VaultAddInput } from '@jterrazz/station-api';
import { useCallback, useRef, useState } from 'react';
import type { RefCallback } from 'react';

import { EngineError } from '../../../domain/engine-error';
import { useClient, useReread } from '../../dependencies/dependencies';
import { VAULT_REREADS } from './passwords-of';

export type Asked = Omit<VaultAddInput, 'Value'>;

export type Adding = {
    secret: RefCallback<HTMLInputElement>;
    sending: boolean;
    refused: EngineError | undefined;
    send: (asked: Asked) => Promise<void>;
    forget: () => void;
};

function refusalOf(error: unknown): EngineError {
    return error instanceof EngineError ? error : new EngineError(undefined, String(error));
}

// invariant: a typed value is read from the element in the frame that sends it, and the element is emptied
// invariant: before the call leaves; with no field drawn (a way that takes no value) the request carries no Value.
function withValue(asked: Asked, element: HTMLInputElement | null): VaultAddInput {
    if (element === null) {
        return asked;
    }
    const typed: VaultAddInput = { ...asked, Value: element.value };
    element.value = '';
    return typed;
}

// invariant: the typed password lives in the uncontrolled input alone: never in state, a store, a log or a
// invariant: recording; the input is emptied on send, refused or not, on close, and when it leaves the page.
export function useAdd(onAdded: () => void): Adding {
    const client = useClient();
    const reread = useReread();
    const held = useRef<HTMLInputElement | null>(null);
    const [sending, setSending] = useState(false);
    const [refused, setRefused] = useState<EngineError | undefined>(undefined);
    const secret = useCallback((element: HTMLInputElement | null) => {
        held.current = element;
        return () => {
            if (element !== null) {
                element.value = '';
            }
            held.current = null;
        };
    }, []);
    const forget = useCallback(() => {
        if (held.current !== null) {
            held.current.value = '';
        }
    }, []);
    const send = async (asked: Asked) => {
        setSending(true);
        setRefused(undefined);
        try {
            await client.call('vault.add', withValue(asked, held.current));
        } catch (error) {
            setRefused(refusalOf(error));
            setSending(false);
            return;
        }
        reread(VAULT_REREADS);
        onAdded();
    };
    return { forget, refused, secret, send, sending };
}
