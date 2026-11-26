import type { CallName, EngineHello, Target } from '@jterrazz/station-api';
import { createContext, useContext, useEffect, useId, useMemo, useSyncExternalStore } from 'react';
import type { ReactNode } from 'react';

import type { CorePort } from '../../application/ports/core.port';
import type { EnginePort, In, Out, ReadName, WatchName } from '../../application/ports/engine.port';
import type { MenuBarPort } from '../../application/ports/menu-bar.port';
import type { OpenerPort } from '../../application/ports/opener.port';
import type { PresencePort } from '../../application/ports/presence.port';
import type { WindowUpdatePort } from '../../application/ports/window-update.port';
import { CoreFlow, NO_CORE } from '../../application/use-cases/core-flow';
import type { CoreState } from '../../application/use-cases/core-flow';
import { READING } from '../../application/use-cases/entry';
import type { Asked, Reading } from '../../application/use-cases/entry';
import { offers } from '../../application/use-cases/fresh';
import { GestureStore } from '../../application/use-cases/gesture-store';
import { MenuBarFlow, NO_MENU_BAR } from '../../application/use-cases/menu-bar-flow';
import type { MenuBarState } from '../../application/use-cases/menu-bar-flow';
import { StateMirror } from '../../application/use-cases/mirror';
import type { MirrorTiming } from '../../application/use-cases/mirror';
import { reachOf } from '../../application/use-cases/reach';
import type { Judged, Reach } from '../../application/use-cases/reach';
import { viewOf } from '../../application/use-cases/sliced';
import { askedKey, ReadStore } from '../../application/use-cases/store';
import type { StoreTiming } from '../../application/use-cases/store';
import { UpdateFlow } from '../../application/use-cases/update-flow';
import type { UpdateState } from '../../application/use-cases/update-flow';
import { NO_WINDOW_UPDATE, WindowUpdateFlow } from '../../application/use-cases/window-update-flow';
import type { WindowUpdateState } from '../../application/use-cases/window-update-flow';
import { Batch } from './batch';
import { StaleMarks } from './stale-marks';
import type { Stale } from './stale-marks';
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
    menuBar?: MenuBarPort | undefined;
    reads?: Partial<StoreTiming> | undefined;
    mirror?: Partial<MirrorTiming> | undefined;
    now?: (() => Date) | undefined;
};

type Engine = {
    batch: Batch;
    client: EnginePort;
    store: ReadStore;
    mirror: StateMirror;
    now: () => Date;
    stale: StaleMarks;
    gestures: GestureStore;
    opener: OpenerPort;
    core: CoreFlow;
    update: UpdateFlow;
    windowUpdate: WindowUpdateFlow;
    menuBar: MenuBarFlow;
};

const EngineContext = createContext<Engine | null>(null);

const NOW = () => new Date();

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
        menuBar = NO_MENU_BAR,
        reads,
        mirror,
        now = NOW,
    } = dependencies;
    const engine = useMemo(() => {
        const mirrored = new StateMirror(client, mirror);
        return {
            batch: new Batch(),
            client,
            core: new CoreFlow(core, coreLookEvery),
            gestures: new GestureStore(),
            menuBar: new MenuBarFlow(menuBar),
            mirror: mirrored,
            now,
            opener,
            stale: new StaleMarks(),
            store: new ReadStore(client, presence, mirrored, reads),
            update: new UpdateFlow(core),
            windowUpdate: new WindowUpdateFlow(windowUpdate),
        };
    }, [client, core, coreLookEvery, menuBar, mirror, now, opener, presence, reads, windowUpdate]);
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

// value: the instant a frame is drawn at, the window's clock: a replay's is the instant its world was recorded at.
export function useNow(): Date {
    return useEngine().now();
}

export function useOpener(): OpenerPort {
    return useEngine().opener;
}

export function useReread(): (names: readonly CallName[]) => void {
    return useEngine().store.reread;
}

function useLink(): Judged | undefined {
    const { store, batch } = useEngine();
    return useSyncExternalStore(batch.of(store.subscribeLink), store.link);
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

const IDLE = () => () => undefined;

// invariant: a view the engine keeps is read from its mirrored stream, never asked by its call, unless the engine
// invariant: could not read it: the call then answers what the window draws, its refusal. Any other read is the store's;
// invariant: nothing asked is read from no one and stays reading.
function useAsked(asked: Asked | undefined): Reading<unknown> {
    const { store, mirror, batch } = useEngine();
    const view = asked === undefined ? undefined : viewOf(asked);
    const subscribe = view === undefined ? IDLE : batch.of(mirror.subscriberOf(view));
    const mirrored = useSyncExternalStore(subscribe, () =>
        view === undefined ? READING : mirror.reading(view),
    );
    const refused = useSyncExternalStore(subscribe, () =>
        view === undefined ? undefined : mirror.refusedOf(view),
    );
    const called =
        asked !== undefined && (view === undefined || refused !== undefined) ? asked : undefined;
    const stored = useSyncExternalStore(called ? batch.of(store.subscriberOf(called)) : IDLE, () =>
        called ? store.reading(askedKey(called)) : READING,
    );
    return called ? stored : mirrored;
}

// invariant: a reader drawn holds its answer's stale refusal in the frame's marks for as long as it is drawn.
function useStaleMark(reading: Reading<unknown>): void {
    const { stale } = useEngine();
    const reader = useId();
    const error = reading.kind === 'read' ? reading.stale : undefined;
    const at = reading.kind === 'read' ? reading.at : undefined;
    useEffect(() => {
        stale.hold(reader, error === undefined || at === undefined ? undefined : { at, error });
    }, [stale, reader, error, at]);
    useEffect(
        () => () => {
            stale.hold(reader, undefined);
        },
        [stale, reader],
    );
}

export function useStale(): readonly Stale[] {
    const { stale, batch } = useEngine();
    return useSyncExternalStore(batch.of(stale.subscribe), stale.shown);
}

// value: what `name` answers for `input`; with no input, nothing is asked and the reading stays reading.
export function useWatch<N extends WatchName>(
    name: N,
    input: In<N> | undefined,
    target?: Target,
    asking: Asking = {},
): Reading<Out<N>> {
    const reading = useAsked(
        input === undefined ? undefined : { input, kind: 'watch', name, target },
    );
    useStaleMark(reading);
    return usePlaceholder(isOutOf(name, reading) ? reading : READING, asking);
}

export function useRead<N extends ReadName>(
    name: N,
    input: In<N>,
    target?: Target,
    asking: Asking = {},
): Reading<Out<N>> {
    const reading = useAsked({ input, kind: 'read', name, target });
    useStaleMark(reading);
    return usePlaceholder(isOutOf(name, reading) ? reading : READING, asking);
}

export function useMenuBar(): { flow: MenuBarFlow; state: MenuBarState } {
    const { menuBar } = useEngine();
    const state = useSyncExternalStore(menuBar.subscribe, menuBar.state);
    return { flow: menuBar, state };
}
