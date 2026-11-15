import type { CallName, Target } from '@jterrazz/station-api';
import { createContext, useContext, useMemo, useSyncExternalStore } from 'react';
import type { ReactNode } from 'react';

import type {
    EnginePort,
    In,
    Link,
    Out,
    ReadName,
    WatchName,
} from '../application/ports/engine.port';
import type { OpenerPort } from '../application/ports/opener.port';
import type { PresencePort } from '../application/ports/presence.port';
import { READING } from '../application/use-cases/entry';
import type { Asked, Reading } from '../application/use-cases/entry';
import { GestureStore } from '../application/use-cases/gesture-store';
import { reachOf } from '../application/use-cases/reach';
import type { Reach } from '../application/use-cases/reach';
import { askedKey, ReadStore } from '../application/use-cases/store';

export type { Reading } from '../application/use-cases/entry';

export type Dependencies = { engine: EnginePort; opener: OpenerPort; presence: PresencePort };

type Engine = { client: EnginePort; store: ReadStore; gestures: GestureStore; opener: OpenerPort };

const EngineContext = createContext<Engine | null>(null);

export function EngineProvider({
    dependencies,
    children,
}: {
    dependencies: Dependencies;
    children: ReactNode;
}) {
    const { engine: client, opener, presence } = dependencies;
    const engine = useMemo(
        () => ({
            client,
            gestures: new GestureStore(),
            opener,
            store: new ReadStore(client, presence),
        }),
        [client, opener, presence],
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

export function useClient(): EnginePort {
    return useEngine().client;
}

export function useGestureStore(): GestureStore {
    return useEngine().gestures;
}

export function useOpener(): OpenerPort {
    return useEngine().opener;
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
