import type { EngineHello } from '@jterrazz/station-api';

import type { Exchange, Recording } from './replay';

function callOf(exchange: Exchange): string[] {
    return typeof exchange.request.call === 'string' ? [exchange.request.call] : [];
}

// value: the hello of a recorded world's engine: it offers the calls the world holds answers for, and no other.
export function helloOf(recording: Recording): EngineHello {
    const exchanges = [...recording.exchanges, ...(recording.gestures ?? [])];
    return {
        calls: [...new Set(exchanges.flatMap(callOf))].toSorted(),
        pid: 0,
        started_at: '',
        version: recording.station,
    };
}
