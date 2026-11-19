import type { MachinesRelease } from '@jterrazz/station-api';

import { keyOf } from '../../../domain/request-key';
import type { Exchange, GestureExchange, Recording } from '../engine-replay/replay';

const READ_RELEASE = { call: 'engine.read-release', in: {} };

const REFRESH_RELEASE = { call: 'engine.refresh-release', in: {} };

// outside: the engine's own answers, copied whole from its stories: the refusal from
// outside: specs/usecases/engine/refresh-release/engine-release-unreachable.spec.yaml, the reading from
// outside: specs/usecases/engine/read-release/latest.spec.yaml. No world records either for the window.
export const REFRESH_REFUSED = {
    code: 'engine.release-unreachable',
    message: 'the latest release was not read: GitHub: dial tcp: connection refused',
    subject: 'release',
    type: 'failed',
} as const;

export const LATEST: MachinesRelease = {
    Behind: false,
    Checked: '2026-11-04T09:00:00Z',
    Head: { Sentence: 'This Mac runs the latest release.', Tone: 'ok' },
    Installed: 'v0.22.0',
    Latest: 'v0.22.0',
    Notes: 'https://github.com/jterrazz/spwn/releases/tag/v0.22.0',
};

const REFUSED: GestureExchange = {
    after: [],
    answers: [{ error: REFRESH_REFUSED, v: 2 }],
    request: REFRESH_RELEASE,
};

export function refreshRefused(recording: Recording): Recording {
    return { ...recording, gestures: [...(recording.gestures ?? []), REFUSED] };
}

const READ_KEY = keyOf(READ_RELEASE);

export function updated(recording: Recording): Recording {
    const latest: Exchange = { answers: [{ out: LATEST, v: 2 }], request: READ_RELEASE };
    return {
        ...recording,
        exchanges: [
            ...recording.exchanges.filter((exchange) => keyOf(exchange.request) !== READ_KEY),
            latest,
        ],
    };
}
