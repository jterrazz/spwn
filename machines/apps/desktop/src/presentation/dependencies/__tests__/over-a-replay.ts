import type { EngineHello } from '@jterrazz/station-api';

import type { Recording } from '../../../infrastructure/outbound/engine-replay/replay';
import { stagedReplay } from '../../../infrastructure/outbound/engine-replay/replay';

export const HELLO = {
    pid: 41,
    started_at: '2026-11-13T09:00:00Z',
    version: 'dev',
};

// value: an engine whose hello offers engine.events, so a notice asks its reads again.
export const NOTICING = { ...HELLO, calls: ['engine.events'] };

// value: a replay of `recording` whose engine is open from the start.
export function opened(recording: Recording, hello: EngineHello = HELLO) {
    return stagedReplay(recording, {
        hello,
        kind: 'open',
        refused: undefined,
        socket: 'replay',
    });
}

export const CHANGES = { characterData: true, childList: true, subtree: true };

// value: every body the page is drawn as while `watcher` observes it.
export function watched() {
    const drawn: string[] = [];
    const watcher = new MutationObserver(() => {
        drawn.push(document.body.innerHTML);
    });
    return { drawn, watcher };
}
