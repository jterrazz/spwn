import type { EngineHello } from '@jterrazz/station-api';

import { PAGE_NEWEST } from '../../../domain/fit';
import type { Exchange, Recording } from './replay';

function callOf(exchange: Exchange): string[] {
    return typeof exchange.request.call === 'string' ? [exchange.request.call] : [];
}

// value: the hello of a recorded world's engine: it offers the calls the world holds answers for, and no other, and
// value: speaks the page revision of the tree that recorded it, the newest this window knows.
export function helloOf(recording: Recording): EngineHello {
    const exchanges = [...recording.exchanges, ...(recording.gestures ?? [])];
    return {
        calls: [...new Set(exchanges.flatMap(callOf))].toSorted(),
        page: PAGE_NEWEST,
        pid: 0,
        started_at: '',
        version: recording.station,
    };
}
