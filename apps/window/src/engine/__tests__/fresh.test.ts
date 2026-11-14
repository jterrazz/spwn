import type { EngineHello, Request } from '@jterrazz/station-api';
import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../client';
import type { Asked } from '../entry';
import type { Presence } from '../presence';
import { ReadStore } from '../store';
import type { Link, Receiver, Transport } from '../transport';

type Opened = { request: Request; receiver: Receiver; closed: boolean };

const HELLO: EngineHello = { pid: 1, started_at: '2026-11-13T09:00:00Z', version: 'v1' };

const EVENTS: EngineHello = { ...HELLO, calls: ['engine.events', 'machines.list'] };

const MACHINES: Asked = { input: {}, kind: 'watch', name: 'machines.list', target: undefined };

const KEY = 'watch machines.list {} {}';

const EVERY = 30_000;

function nameOf(request: Request): string | undefined {
    if ('watch' in request) {
        return request.watch;
    }
    return 'call' in request ? request.call : undefined;
}

function fakeEngine(hello: EngineHello) {
    const opened: Opened[] = [];
    const transport: Transport = {
        link: (listener) => {
            const link: Link = { hello, kind: 'open', refused: undefined, socket: 'fake' };
            const timer = setTimeout(() => {
                listener(link);
            }, 0);
            return () => {
                clearTimeout(timer);
            };
        },
        open: (request, receiver) => {
            const entry = { closed: false, receiver, request };
            opened.push(entry);
            return {
                cancel: () => undefined,
                close: () => {
                    entry.closed = true;
                },
            };
        },
    };
    const of = (name: string) => opened.filter((one) => nameOf(one.request) === name);
    const answer = (name: string, ...frames: Record<string, unknown>[]) => {
        const one = of(name).findLast((each) => !each.closed);
        for (const frame of frames) {
            one?.receiver.frame({ ...frame, id: one.request.id, v: 2 });
        }
    };
    return { answer, asked: (name: string) => of(name).length, transport };
}

function fakePresence() {
    let visible = true;
    const listeners = new Set<(seen: 'back' | 'hidden') => void>();
    const presence: Presence = {
        subscribe: (listener) => {
            listeners.add(listener);
            return () => {
                listeners.delete(listener);
            };
        },
        visible: () => visible,
    };
    const set = (seen: 'back' | 'hidden') => {
        visible = seen === 'back';
        for (const listener of listeners) {
            listener(seen);
        }
    };
    return { listening: () => listeners.size, presence, set };
}

const RUNNING = { out: { State: 'running' } };
const PENDING = { out: { State: 'pending' } };
const STOPPED = { out: { State: 'stopped' } };
const DONE = { done: true };

async function settled(hello: EngineHello = HELLO) {
    const engine = fakeEngine(hello);
    const seen = fakePresence();
    const store = new ReadStore(
        new Client(engine.transport),
        { every: EVERY, grace: 500 },
        seen.presence,
    );
    const leave = store.subscribe(MACHINES, () => {});
    await clock.advance(1);
    engine.answer('machines.list', RUNNING, DONE);
    return { engine, leave, seen, store };
}

describe('fresh', () => {
    test('a watch that ended is asked again on the cadence, and shows what the engine now says', async () => {
        // Given - machines.list answered running, then ended
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();

        // When - the VM is stopped from outside the window and the cadence passes
        await clock.advance(EVERY);
        engine.answer('machines.list', STOPPED, DONE);

        // Then - the watch was asked twice and shows the stopped state
        expect(engine.asked('machines.list')).toBe(2);
        expect(store.reading(KEY)).toStrictEqual({ done: true, kind: 'read', out: STOPPED.out });
    });

    test('a re-ask keeps the shown answer while the engine sends pending values, then shows its last', async () => {
        // Given - machines.list settled on running
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();
        await clock.advance(EVERY);

        // When - the re-ask answers pending first
        engine.answer('machines.list', PENDING);
        const meanwhile = store.reading(KEY);
        engine.answer('machines.list', STOPPED, DONE);

        // Then - the pending value was never shown, nor a reading state
        expect(meanwhile).toStrictEqual({ done: true, kind: 'read', out: RUNNING.out });
        expect(store.reading(KEY)).toStrictEqual({ done: true, kind: 'read', out: STOPPED.out });
    });

    test('a re-ask still unanswered at the next tick is not asked again, and its last value is shown', async () => {
        // Given - a re-ask that answers pending and never ends
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();
        await clock.advance(EVERY);
        engine.answer('machines.list', PENDING);

        // When - two more cadences pass
        await clock.advance(EVERY * 2);

        // Then - one re-ask in all, and what it said is shown rather than held for ever
        expect(engine.asked('machines.list')).toBe(2);
        expect(store.reading(KEY)).toStrictEqual({ done: false, kind: 'read', out: PENDING.out });
    });

    test('a watch still streaming is never asked again', async () => {
        // Given - a watch that answered and has not ended
        using _ = clock.run('2026-11-13T10:00:00Z');
        const engine = fakeEngine(HELLO);
        const seen = fakePresence();
        const store = new ReadStore(new Client(engine.transport), { every: EVERY }, seen.presence);
        store.subscribe(MACHINES, () => {});
        await clock.advance(1);
        engine.answer('machines.list', RUNNING);

        // When - three cadences pass
        await clock.advance(EVERY * 3);

        // Then - its one request stays open
        expect(engine.asked('machines.list')).toBe(1);
    });

    test('nothing is asked again once nobody subscribes', async () => {
        // Given - machines.list settled, then left by its only subscriber
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, leave, seen } = await settled();
        leave();

        // When - the cadence passes within the grace, then long after it
        await clock.advance(400);
        await clock.advance(EVERY * 3);

        // Then - no re-ask, and the window's presence is no longer watched
        expect(engine.asked('machines.list')).toBe(1);
        expect(seen.listening()).toBe(0);
    });

    test('the cadence pauses while the window is hidden, and every read is asked once it is shown again', async () => {
        // Given - machines.list settled, then the window hidden
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, seen } = await settled();
        seen.set('hidden');

        // When - three cadences pass hidden, then the window is shown
        await clock.advance(EVERY * 3);
        const whileHidden = engine.asked('machines.list');
        seen.set('back');

        // Then - nothing was asked while hidden, and one ask came at once on return
        expect(whileHidden).toBe(1);
        expect(engine.asked('machines.list')).toBe(2);
    });

    test('a window focused again asks at once, and focus twice in a row asks once', async () => {
        // Given - machines.list settled
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, seen } = await settled();

        // When - the window regains the focus twice before the engine answers
        seen.set('back');
        seen.set('back');

        // Then - one re-ask, not two
        expect(engine.asked('machines.list')).toBe(2);
    });

    test('an engine that offers engine.events has its reads asked again on a notice', async () => {
        // Given - an engine whose hello lists engine.events, and machines.list settled
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled(EVENTS);
        engine.answer('engine.events', { out: { stream: 'engine.events' } });
        const opened = engine.asked('machines.list');

        // When - the engine tells a notice, well before the cadence
        engine.answer('engine.events', {
            out: { event: 'notice', notice: { kind: 'vm_ready', text: 'api is ready' } },
        });
        engine.answer('machines.list', STOPPED, DONE);

        // Then - the opening said nothing, the notice asked once, and the new answer shows
        expect(opened).toBe(1);
        expect(engine.asked('engine.events')).toBe(1);
        expect(engine.asked('machines.list')).toBe(2);
        expect(store.reading(KEY)).toStrictEqual({ done: true, kind: 'read', out: STOPPED.out });
    });

    test('an engine that does not offer engine.events is never asked it, and one that refuses it keeps the cadence', async () => {
        // Given - one engine without engine.events in its hello, one whose hello lists it and refuses it
        using _ = clock.run('2026-11-13T10:00:00Z');
        const plain = await settled();
        const refusing = await settled(EVENTS);
        refusing.engine.answer('engine.events', {
            error: { code: 'call.unknown', message: 'no call engine.events', type: 'refused' },
        });

        // When - the cadence passes and a backoff's worth of time after it
        await clock.advance(EVERY + 5000);

        // Then - the first never asked it, the second asked it once, and both re-read machines.list
        expect(plain.engine.asked('engine.events')).toBe(0);
        expect(refusing.engine.asked('engine.events')).toBe(1);
        expect([
            plain.engine.asked('machines.list'),
            refusing.engine.asked('machines.list'),
        ]).toStrictEqual([2, 2]);
    });
});
