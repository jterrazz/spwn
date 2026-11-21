import { Channel, invoke } from '@tauri-apps/api/core';

import type { CorePort } from '../application/ports/core.port';
import type { Link } from '../application/ports/engine.port';
import type { OpenerPort } from '../application/ports/opener.port';
import type { WindowUpdatePort } from '../application/ports/window-update.port';
import { Launch } from '../application/use-cases/launch';
import { tauriCore } from '../infrastructure/outbound/core-tauri/tauri-core';
import { documentPresence } from '../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../infrastructure/outbound/engine-link/client';
import type { Transport } from '../infrastructure/outbound/engine-link/transport';
import { helloOf } from '../infrastructure/outbound/engine-replay/hello';
import { replayTransport, stagedReplay } from '../infrastructure/outbound/engine-replay/replay';
import type { Pace, Recording } from '../infrastructure/outbound/engine-replay/replay';
import { documentLaunch } from '../infrastructure/outbound/launch-document/document-launch';
import { NO_SHOW, tauriShow } from '../infrastructure/outbound/launch-tauri/tauri-show';
import { tauriTransport } from '../infrastructure/outbound/engine-tauri/tauri';
import { NO_OPENER, tauriOpener } from '../infrastructure/outbound/opener-tauri/tauri-opener';
import { tauriWindowUpdate } from '../infrastructure/outbound/window-update-tauri/tauri-window-update';
import type { Dependencies } from '../presentation/dependencies/dependencies';

// invariant: a build-time constant, so a production build drops the recorded worlds' import below.
const PROOF = import.meta.env.MODE !== 'production';

export type Place = { inTauri: boolean; query: URLSearchParams };

export type Replayed = {
    recording: Recording;
    link?: Link | undefined;
    pace?: Pace | undefined;
    opener?: OpenerPort | undefined;
    core?: CorePort | undefined;
};

const NO_ENGINE_YET: Link = {
    kind: 'absent',
    reason: 'connection refused',
    retryIn: 30_000,
    socket: '/Users/you/.local/state/station/engine.sock',
};

function absent(reason: string): Transport {
    return replayTransport(
        { exchanges: [], station: '', world: 'none' },
        { kind: 'absent', reason, retryIn: 30_000, socket: 'no socket' },
    );
}

type Composed = { transport: Transport; core?: CorePort | undefined; coreLookEvery?: number };

const SCRIPTED_LOOK_EVERY = 500;

async function scriptedOf(recording: Recording, script: string, pace: Pace): Promise<Composed> {
    const { scriptOf } = await import('../infrastructure/outbound/core-scripted/scripts');
    const found = scriptOf(script);
    if (found === undefined) {
        return { transport: replayTransport(recording, undefined, pace) };
    }
    const { scriptedCore } = await import('../infrastructure/outbound/core-scripted/scripted-core');
    if (found.answering === true) {
        const { updated } = await import('../infrastructure/outbound/core-scripted/release');
        const stage = stagedReplay(recording, undefined, pace);
        const after = updated(recording);
        const core = scriptedCore(found, {
            gap: pace.gestureFrame,
            onStarted: () => {
                stage.restart({ ...helloOf(after), pid: 1, started_at: 'restarted' }, after);
            },
        });
        return { core, transport: stage.transport };
    }
    const stage = stagedReplay(recording, NO_ENGINE_YET, pace);
    const opened: Link = {
        hello: undefined,
        kind: 'open',
        refused: undefined,
        socket: `replay of ${recording.world}`,
    };
    const core = scriptedCore(found, {
        gap: pace.gestureFrame,
        onStarted: () => {
            stage.show(opened);
        },
    });
    return { core, coreLookEvery: SCRIPTED_LOOK_EVERY, transport: stage.transport };
}

async function composedOf({ inTauri, query }: Place): Promise<Composed> {
    if (inTauri) {
        return {
            core: tauriCore({ channel: (heard) => new Channel(heard), invoke }),
            transport: tauriTransport(),
        };
    }
    const world = query.get('world') ?? 'two-macs';
    if (!PROOF || world === 'none') {
        return { transport: absent('no engine bridge outside the Tauri window') };
    }
    const { recordingFor } = await import('../infrastructure/outbound/engine-replay/recordings');
    const pace = {
        gestureFrame: Number(query.get('pace') ?? '0'),
        held: query.get('held') ?? undefined,
    };
    const recorded = await recordingFor(world);
    const release = await import('../infrastructure/outbound/core-scripted/release');
    const recording =
        query.get('release') === 'refused' ? release.refreshRefused(recorded) : recorded;
    const script = query.get('core');
    if (script !== null) {
        return await scriptedOf(recording, script, pace);
    }
    if (query.get('link') === 'blip') {
        const { blippedReplay } = await import('../infrastructure/outbound/engine-replay/blip');
        return { transport: blippedReplay(recording, pace) };
    }
    return { transport: replayTransport(recording, undefined, pace) };
}

async function windowUpdateOf({ inTauri, query }: Place): Promise<undefined | WindowUpdatePort> {
    if (inTauri) {
        return tauriWindowUpdate({ channel: (heard) => new Channel(heard), invoke });
    }
    const name = query.get('window');
    if (!PROOF || name === null) {
        return undefined;
    }
    const scripted =
        await import('../infrastructure/outbound/window-update-scripted/scripted-window-update');
    const script = scripted.windowScriptOf(name);
    return script === undefined
        ? undefined
        : scripted.scriptedWindowUpdate(script, Number(query.get('pace') ?? '0'));
}

export async function compose(place: Place): Promise<Dependencies> {
    const { core, coreLookEvery, transport } = await composedOf(place);
    return {
        core,
        coreLookEvery,
        engine: new Client(transport),
        opener: place.inTauri ? tauriOpener({ invoke }) : NO_OPENER,
        presence: documentPresence(),
        windowUpdate: await windowUpdateOf(place),
    };
}

export function composeLaunch({ inTauri }: Pick<Place, 'inTauri'>): Launch {
    return new Launch(documentLaunch(inTauri ? tauriShow({ invoke }) : NO_SHOW));
}

export function composeReplay({
    recording,
    link,
    pace,
    opener = NO_OPENER,
    core,
}: Replayed): Dependencies {
    return {
        core,
        engine: new Client(replayTransport(recording, link, pace)),
        opener,
        presence: documentPresence(),
    };
}

export function composeGallerySample(): Dependencies {
    return composeReplay({ recording: { exchanges: [], station: 'gallery', world: 'gallery' } });
}
