import type { CallName, Target } from '@jterrazz/station-api';
import { useEffect, useRef, useSyncExternalStore } from 'react';

import type { Client, In, Out, ReadName } from './client';
import { isConfirmed } from './confirms';
import { EngineError } from './engine-error';
import { keyOf } from './frame';
import { IDLE, step } from './gesture';
import type { Ask, GestureEvent, GestureState } from './gesture';
import { confirmFormOf } from './provisional';
import { useClient, useGestureStore, useReread } from './react';

export type GestureSpec<N extends ReadName> = {
    input: In<N>;
    subject: string;
    rereads: readonly CallName[];
    target?: Target | undefined;
    onDone?: (out: Out<N>) => void;
    within?: number;
};

// value: how long a gesture may go without a frame before its outcome is unknown: a gesture's work runs long.
const GESTURE_WITHIN = 10 * 60_000;

export type Gesture<O> = {
    key: string;
    state: GestureState<O>;
    start: () => void;
    confirm: (typed: string) => void;
    cancel: () => void;
    dismiss: () => void;
};

function firstAsk(name: CallName, subject: string): Ask | undefined {
    return isConfirmed(name)
        ? { form: confirmFormOf(name), name: subject, refusal: undefined }
        : undefined;
}

type Maker<N extends ReadName> = {
    client: Client;
    name: N;
    send: (event: GestureEvent<Out<N>>) => void;
};

async function made<N extends ReadName>(
    { client, name, send }: Maker<N>,
    spec: GestureSpec<N>,
    answer: string | undefined,
): Promise<void> {
    const input = answer === undefined ? spec.input : { ...spec.input, Confirm: answer };
    try {
        const out = await client.call(name, input, {
            progress: (progress) => {
                send({ kind: 'progress', progress });
            },
            target: spec.target,
            within: spec.within ?? GESTURE_WITHIN,
        });
        send({ kind: 'answered', out });
    } catch (error) {
        const failed =
            error instanceof EngineError ? error : new EngineError(undefined, String(error));
        send({ error: failed, kind: failed.fate === 'lost' ? 'lost' : 'refused' });
    }
}

// invariant: the store holds what `step` made of this call's events, so its out is the call's.
function isStateOf<N extends ReadName>(
    _name: N,
    state: GestureState<unknown>,
): state is GestureState<Out<N>> {
    return state.kind !== 'done' || state.out !== undefined;
}

export function useGesture<N extends ReadName>(name: N, spec: GestureSpec<N>): Gesture<Out<N>> {
    const client = useClient();
    const reread = useReread();
    const gestures = useGestureStore();
    const key = keyOf({ call: name, in: spec.input, target: spec.target });
    const held = useSyncExternalStore(gestures.subscriberOf(key), () => gestures.state(key));
    const state = isStateOf(name, held) ? held : IDLE;
    const latest = useRef(spec);
    useEffect(() => {
        latest.current = spec;
    });
    function send(event: GestureEvent<Out<N>>) {
        const now = gestures.state(key);
        const before = isStateOf(name, now) ? now : IDLE;
        const after = step(before, event);
        gestures.set(key, after);
        if (before.kind !== 'running' && after.kind === 'running') {
            void made({ client, name, send }, latest.current, after.answer);
        }
        if (before.kind === 'running' && after.kind === 'done') {
            latest.current.onDone?.(after.out);
        }
        if (before.kind === 'running' && after.kind !== 'running' && after.kind !== 'asking') {
            reread(latest.current.rereads);
        }
    }
    return {
        cancel: () => {
            send({ kind: 'cancel' });
        },
        confirm: (typed) => {
            send({ answer: typed, kind: 'confirm' });
        },
        dismiss: () => {
            send({ kind: 'dismiss' });
        },
        key,
        start: () => {
            send({ ask: firstAsk(name, latest.current.subject), kind: 'start' });
        },
        state,
    };
}
