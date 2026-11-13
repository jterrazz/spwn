import type { CallName, Target } from '@jterrazz/station-api';
import { createContext, useContext, useMemo, useSyncExternalStore } from 'react';
import type { ReactNode } from 'react';

import type { Client, In, Out, ReadName, WatchName } from './client';
import { READING } from './entry';
import type { Asked, Reading } from './entry';
import { GestureStore } from './gesture-store';
import { reachOf } from './reach';
import type { Reach } from './reach';
import { askedKey, ReadStore } from './store';
import type { Link } from './transport';

export type { Reading } from './entry';

type Engine = { client: Client; store: ReadStore; gestures: GestureStore };

const EngineContext = createContext<Engine | null>(null);

export function EngineProvider({ client, children }: { client: Client; children: ReactNode }) {
    const engine = useMemo(
        () => ({ client, gestures: new GestureStore(), store: new ReadStore(client) }),
        [client],
    );
    return <EngineContext value={engine}>{children}</EngineContext>;
}

function useEngine(): Engine {
    const engine = useContext(EngineContext);
    if (!engine) {
        throw new Error('The engine is read inside an EngineProvider.');
    }
    return engine;
}

export function useClient(): Client {
    return useEngine().client;
}

export function useGestureStore(): GestureStore {
    return useEngine().gestures;
}

export function useReread(): (names: readonly CallName[]) => void {
    return useEngine().store.reread;
}

function useLink(): Link | undefined {
    const { store } = useEngine();
    return useSyncExternalStore(store.subscribeLink, store.link);
}

export function useReach(): Reach {
    return reachOf(useLink());
}

// invariant: the answer is the type the call declares: the store holds what the client resolved for this name.
function isOutOf<N extends CallName>(
    _name: N,
    reading: Reading<unknown>,
): reading is Reading<Out<N>> {
    return reading.kind !== 'read' || reading.out !== undefined;
}

function useAsked(asked: Asked): Reading<unknown> {
    const { store } = useEngine();
    const key = askedKey(asked);
    return useSyncExternalStore(store.subscriberOf(asked), () => store.reading(key));
}

export function useWatch<N extends WatchName>(
    name: N,
    input: In<N>,
    target?: Target,
): Reading<Out<N>> {
    const reading = useAsked({ input, kind: 'watch', name, target });
    return isOutOf(name, reading) ? reading : READING;
}

export function useRead<N extends ReadName>(
    name: N,
    input: In<N>,
    target?: Target,
): Reading<Out<N>> {
    const reading = useAsked({ input, kind: 'read', name, target });
    return isOutOf(name, reading) ? reading : READING;
}
