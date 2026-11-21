import type { CallName, EngineHello, Target } from '@jterrazz/station-api';
import { createContext, useContext, useEffect, useMemo, useSyncExternalStore } from 'react';
import type { ReactNode } from 'react';

import type { CorePort } from '../../application/ports/core.port';
import type {
    EnginePort,
    In,
    Link,
    Out,
    ReadName,
    WatchName,
} from '../../application/ports/engine.port';
import type { OpenerPort } from '../../application/ports/opener.port';
import type { PresencePort } from '../../application/ports/presence.port';
import type { WindowUpdatePort } from '../../application/ports/window-update.port';
import { CoreFlow, NO_CORE } from '../../application/use-cases/core-flow';
import type { CoreState } from '../../application/use-cases/core-flow';
import { READING } from '../../application/use-cases/entry';
import type { Asked, Reading } from '../../application/use-cases/entry';
import { offers } from '../../application/use-cases/fresh';
import { GestureStore } from '../../application/use-cases/gesture-store';
import { reachOf } from '../../application/use-cases/reach';
import type { Reach } from '../../application/use-cases/reach';
import { askedKey, ReadStore } from '../../application/use-cases/store';
import type { StoreTiming } from '../../application/use-cases/store';
import { UpdateFlow } from '../../application/use-cases/update-flow';
import type { UpdateState } from '../../application/use-cases/update-flow';
import { NO_WINDOW_UPDATE, WindowUpdateFlow } from '../../application/use-cases/window-update-flow';
import type { WindowUpdateState } from '../../application/use-cases/window-update-flow';
import { usePlaceholder } from './use-placeholder';
import type { Asking } from './use-placeholder';

export type { Reading } from '../../application/use-cases/entry';
export type { Asking } from './use-placeholder';

export type Dependencies = {
    engine: EnginePort;
    opener: OpenerPort;
    presence: PresencePort;
    core?: CorePort | undefined;
    coreLookEvery?: number | undefined;
    windowUpdate?: undefined | WindowUpdatePort;
    reads?: Partial<StoreTiming> | undefined;
};

type Engine = {
    client: EnginePort;
    store: ReadStore;
    gestures: GestureStore;
    opener: OpenerPort;
    core: CoreFlow;
    update: UpdateFlow;
    windowUpdate: WindowUpdateFlow;
};

const EngineContext = createContext<Engine | null>(null);

export function EngineProvider({
    dependencies,
    children,
}: {
    dependencies: Dependencies;
    children: ReactNode;
}) {
    const {
        engine: client,
        opener,
        presence,
        core = NO_CORE,
        coreLookEvery,
        windowUpdate = NO_WINDOW_UPDATE,
        reads,
    } = dependencies;
    const engine = useMemo(
        () => ({
            client,
            core: new CoreFlow(core, coreLookEvery),
            gestures: new GestureStore(),
            opener,
            store: new ReadStore(client, presence, reads),
            update: new UpdateFlow(core),
            windowUpdate: new WindowUpdateFlow(windowUpdate),
        }),
        [client, core, coreLookEvery, opener, presence, reads, windowUpdate],
    );
    useEffect(() => {
        engine.windowUpdate.start();
        return () => {
            engine.core.close();
        };
    }, [engine]);
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

export function useCore(): { flow: CoreFlow; state: CoreState } {
    const { core } = useEngine();
    const state = useSyncExternalStore(core.subscribe, core.state);
    return { flow: core, state };
}

export function useUpdate(): { flow: UpdateFlow; state: UpdateState } {
    const { update } = useEngine();
    const state = useSyncExternalStore(update.subscribe, update.state);
    return { flow: update, state };
}

export function useWindowUpdate(): { flow: WindowUpdateFlow; state: WindowUpdateState } {
    const { windowUpdate } = useEngine();
    const state = useSyncExternalStore(windowUpdate.subscribe, windowUpdate.state);
    return { flow: windowUpdate, state };
}

export function useHello(): EngineHello | undefined {
    const link = useLink();
    return link?.kind === 'open' ? link.hello : undefined;
}

export function useOffers(call: CallName): boolean {
    return offers(useHello(), call);
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
    asking: Asking = {},
): Reading<Out<N>> {
    const reading = useAsked({ input, kind: 'watch', name, target });
    return usePlaceholder(isOutOf(name, reading) ? reading : READING, asking);
}

export function useRead<N extends ReadName>(
    name: N,
    input: In<N>,
    target?: Target,
    asking: Asking = {},
): Reading<Out<N>> {
    const reading = useAsked({ input, kind: 'read', name, target });
    return usePlaceholder(isOutOf(name, reading) ? reading : READING, asking);
}
