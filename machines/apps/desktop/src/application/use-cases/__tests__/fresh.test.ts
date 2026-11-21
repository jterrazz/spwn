import type { EngineHello, Request } from '@jterrazz/station-api';
import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../../infrastructure/outbound/engine-link/client';
import type { Receiver, Transport } from '../../../infrastructure/outbound/engine-link/transport';
import type { Link } from '../../ports/engine.port';
import type { PresencePort } from '../../ports/presence.port';
import type { Asked, Reading } from '../entry';
import { offers } from '../fresh';
import { ReadStore } from '../store';

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
    const end = (name: string) => {
        of(name)
            .findLast((each) => !each.closed)
            ?.receiver.end('the engine closed the connection', true);
    };
    return { answer, asked: (name: string) => of(name).length, end, transport };
}

function fakePresence() {
    let visible = true;
    const listeners = new Set<(seen: 'back' | 'hidden') => void>();
    const presence: PresencePort = {
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
const UNREACHABLE = {
    code: 'remote.unreachable',
    message: 'studio did not answer',
    type: 'unreachable',
};

async function settled(hello: EngineHello = HELLO) {
    const engine = fakeEngine(hello);
    const seen = fakePresence();
    const store = new ReadStore(new Client(engine.transport), seen.presence, {
        every: EVERY,
        grace: 500,
    });
    let told = 0;
    const leave = store.subscribe(MACHINES, () => {
        told += 1;
    });
    await clock.advance(1);
    engine.answer('machines.list', RUNNING, DONE);
    return { engine, heard: () => told, leave, seen, store };
}

function rowsOf(reading: Reading<unknown>): readonly unknown[] {
    return reading.kind === 'read' && Array.isArray(reading.out) ? reading.out : [];
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

    test('a re-ask answered equal keeps the reading itself and tells no subscriber', async () => {
        // Given - machines.list settled on running, and what its subscriber has heard so far
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, heard, store } = await settled();
        const shown = store.reading(KEY);
        const told = heard();

        // When - the cadence passes and the engine answers the same state, as a new object
        await clock.advance(EVERY);
        engine.answer('machines.list', { out: { State: 'running' } }, DONE);

        // Then - it was asked again, and the reading is the one shown, its subscriber never told
        expect(engine.asked('machines.list')).toBe(2);
        expect(store.reading(KEY)).toBe(shown);
        expect(heard()).toBe(told);
    });

    test('a re-ask that moves one row shows a new reading whose other rows are the shown ones', async () => {
        // Given - machines.list settled on two running VMs
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();
        const rows = [
            { Name: 'api', State: 'running' },
            { Name: 'web', State: 'running' },
        ];
        await clock.advance(EVERY);
        engine.answer('machines.list', { out: rows }, DONE);
        const [, web] = rowsOf(store.reading(KEY));

        // When - the next cadence answers api stopped and web as it was
        await clock.advance(EVERY);
        engine.answer(
            'machines.list',
            { out: [{ ...rows[0], State: 'stopped' }, { ...rows[1] }] },
            DONE,
        );

        // Then - the reading shows api stopped, and web is the very row shown before
        const [api, again] = rowsOf(store.reading(KEY));
        expect(api).toStrictEqual({ Name: 'api', State: 'stopped' });
        expect(again).toBe(web);
    });

    test('a re-ask a far Mac misses keeps the shown answer marked stale, and the next answer clears the mark', async () => {
        // Given - machines.list settled on running
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();

        // When - the next re-ask is refused as unreachable, then the one after answers again
        await clock.advance(EVERY);
        engine.answer('machines.list', { error: UNREACHABLE });
        const missed = store.reading(KEY);
        await clock.advance(EVERY);
        engine.answer('machines.list', RUNNING, DONE);

        // Then - the answer stayed shown with the refusal as its stale mark, and the mark went with the new answer
        expect(missed.kind === 'read' ? [missed.out, missed.stale?.wire?.type] : []).toStrictEqual([
            RUNNING.out,
            'unreachable',
        ]);
        expect(store.reading(KEY)).toStrictEqual({ done: true, kind: 'read', out: RUNNING.out });
    });

    test('a re-ask whose connection drops keeps the shown answer marked stale, and a refusal of the subject replaces it', async () => {
        // Given - two stores, each with machines.list settled on running
        using _ = clock.run('2026-11-13T10:00:00Z');
        const dropped = await settled();
        const refused = await settled();

        // When - one re-ask's connection ends unanswered, the other is answered that the machine is not found
        await clock.advance(EVERY);
        dropped.engine.end('machines.list');
        refused.engine.answer('machines.list', {
            error: { code: 'unknown_vm', message: 'no such machine', type: 'not-found' },
        });

        // Then - the first keeps its answer with a stale mark of no engine word, the second shows the refusal
        const kept = dropped.store.reading(KEY);
        expect(kept.kind === 'read' ? [kept.out, kept.stale?.wire] : []).toStrictEqual([
            RUNNING.out,
            undefined,
        ]);
        expect(refused.store.reading(KEY).kind).toBe('failed');
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
        const store = new ReadStore(new Client(engine.transport), seen.presence, { every: EVERY });
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

    test('a call is offered only when the hello lists it, and no hello offers nothing', () => {
        // Given - a hello listing the release reading, one without calls, and none
        const listing: EngineHello = {
            calls: ['engine.read-release'],
            pid: 1,
            started_at: '',
            version: 'v0.22.0',
        };
        const bare: EngineHello = { pid: 1, started_at: '', version: 'v0.20.0' };

        // When - each is asked for the release reading
        const offered = [listing, bare, undefined].map((hello) =>
            offers(hello, 'engine.read-release'),
        );

        // Then - the listing hello alone offers it
        expect(offered).toStrictEqual([true, false, false]);
    });
});
