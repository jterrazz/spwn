import { invoke } from '@tauri-apps/api/core';

import type { Link } from '../application/ports/engine.port';
import type { OpenerPort } from '../application/ports/opener.port';
import { documentPresence } from '../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../infrastructure/outbound/engine-link/client';
import type { Transport } from '../infrastructure/outbound/engine-link/transport';
import { replayTransport } from '../infrastructure/outbound/engine-replay/replay';
import type { Pace, Recording } from '../infrastructure/outbound/engine-replay/replay';
import { tauriTransport } from '../infrastructure/outbound/engine-tauri/tauri';
import { NO_OPENER, tauriOpener } from '../infrastructure/outbound/opener-tauri/tauri-opener';
import type { Dependencies } from '../presentation/dependencies/dependencies';

// invariant: a build-time constant, so a production build drops the recorded worlds' import below.
const PROOF = import.meta.env.MODE !== 'production';

export type Place = { inTauri: boolean; query: URLSearchParams };

export type Replayed = {
    recording: Recording;
    link?: Link | undefined;
    pace?: Pace | undefined;
    opener?: OpenerPort | undefined;
};

function absent(reason: string): Transport {
    return replayTransport(
        { exchanges: [], station: '', world: 'none' },
        { kind: 'absent', reason, retryIn: 30_000, socket: 'no socket' },
    );
}

async function transportOf({ inTauri, query }: Place): Promise<Transport> {
    if (inTauri) {
        return tauriTransport();
    }
    const world = query.get('world') ?? 'two-macs';
    if (!PROOF || world === 'none') {
        return absent('no engine bridge outside the Tauri window');
    }
    const { recordingFor } = await import('../infrastructure/outbound/engine-replay/recordings');
    const pace = { gestureFrame: Number(query.get('pace') ?? '0') };
    return replayTransport(await recordingFor(world), undefined, pace);
}

export async function compose(place: Place): Promise<Dependencies> {
    return {
        engine: new Client(await transportOf(place)),
        opener: place.inTauri ? tauriOpener({ invoke }) : NO_OPENER,
        presence: documentPresence(),
    };
}

export function composeReplay({
    recording,
    link,
    pace,
    opener = NO_OPENER,
}: Replayed): Dependencies {
    return {
        engine: new Client(replayTransport(recording, link, pace)),
        opener,
        presence: documentPresence(),
    };
}

export function composeGallerySample(): Dependencies {
    return composeReplay({ recording: { exchanges: [], station: 'gallery', world: 'gallery' } });
}
