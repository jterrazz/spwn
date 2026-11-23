import { clock, waitUntil } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { documentPresence } from '../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../infrastructure/outbound/engine-link/client';
import { stagedReplay } from '../../../infrastructure/outbound/engine-replay/replay';
import type { Pace, Recording } from '../../../infrastructure/outbound/engine-replay/replay';
import type { Asked } from '../entry';
import { StateMirror } from '../mirror';
import { ReadStore } from '../store';
import { fakeEngine, fakePresence, HELLO, settled } from './fake-engine';

const READ = { call: 'engine.read', in: {} };
const MACHINES = { in: {}, watch: 'machines.list' };
const ON_API = { call: 'engine.read', in: {}, target: { name: 'api', on: 'mac' } };

const RUNNING: Recording = {
    exchanges: [
        { answers: [{ out: { State: 'running' }, v: 2 }], request: READ },
        { answers: [{ out: [{ Name: 'api' }], v: 2 }], request: MACHINES },
        { answers: [{ out: { State: 'running', Version: 'api' }, v: 2 }], request: ON_API },
    ],
    gestures: [
        {
            after: [{ answers: [{ out: { State: 'stopped' }, v: 2 }], request: READ }],
            answers: [{ out: {}, v: 2 }],
            request: { call: 'engine.stop', in: { Confirm: 'yes' } },
        },
    ],
    station: 'dev',
    world: 'test',
};

const RESTARTED: Recording = {
    ...RUNNING,
    exchanges: [{ answers: [{ out: { State: 'running', Version: 'v2' }, v: 2 }], request: READ }],
};

const API_KEY = 'call engine.read {} {"name":"api","on":"mac"}';
const READ_KEY = 'call engine.read {} {}';
const MACHINES_KEY = 'watch machines.list {} {}';

const MACHINES_WATCH: Asked = {
    input: {},
    kind: 'watch',
    name: 'machines.list',
    target: undefined,
};

const KEPT_FOR = 10_000;

function engineRead(input: Record<string, never> = {}): Asked {
    return { input, kind: 'read', name: 'engine.read', target: undefined };
}

function staged(pace?: Pace) {
    const stage = stagedReplay(
        RUNNING,
        { hello: HELLO, kind: 'open', refused: undefined, socket: 'replay of test' },
        pace,
    );
    const client = new Client(stage.transport);
    const timing = { grace: 500, keptAtMost: 2, keptFor: KEPT_FOR };
    const store = new ReadStore(client, documentPresence(), new StateMirror(client), timing);
    return { client, stage, store };
}

function asked(stage: ReturnType<typeof stagedReplay>, call: string): number {
    return stage.asked().filter((request) => 'call' in request && request.call === call).length;
}

describe('store', () => {
    test('two subscribers of an equal request share one entry and one exchange', async () => {
        // Given - a store over a replay with 50 ms of latency
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { stage, store } = staged({ gestureFrame: 0, latency: () => 50 });
        const heard: string[] = [];

        // When - two components subscribe to api's engine.read, each with its own objects, the target spelled apart
        store.subscribe({ ...engineRead(), target: { name: 'api', on: 'mac' } }, () => {
            heard.push('first');
        });
        store.subscribe({ ...engineRead({}), target: { on: 'mac', name: 'api' } }, () => {
            heard.push('second');
        });
        await clock.advance(10);
        const whileWaiting = store.reading(API_KEY).kind;
        await clock.advance(50);

        // Then - one request went out, both heard its answer, and the latency showed as reading
        expect(asked(stage, 'engine.read')).toBe(1);
        expect(stage.open()).toBe(0);
        expect(whileWaiting).toBe('reading');
        expect(heard).toStrictEqual(['first', 'second']);
        expect(store.reading(API_KEY)).toStrictEqual({
            at: new Date('2026-11-13T10:00:00.050Z'),
            done: true,
            kind: 'read',
            out: { State: 'running', Version: 'api' },
        });
    });

    test('the last subscriber leaving closes the watch after its grace, its answer kept until its time is spent', async () => {
        // Given - a watch of machines.list held by one subscriber, streaming
        using _ = clock.run('2026-11-13T10:00:00Z');
        const engine = fakeEngine(HELLO);
        const client = new Client(engine.transport);
        const store = new ReadStore(client, fakePresence().presence, new StateMirror(client), {
            grace: 500,
            keptFor: KEPT_FOR,
        });
        const leave = store.subscribe(MACHINES_WATCH, () => {});
        await clock.advance(1);
        engine.answer('machines.list', { out: [{ Name: 'api' }] });

        // When - it leaves and comes back within the grace, leaves for longer than it, then past its keeping
        leave();
        await clock.advance(400);
        const back = store.subscribe(MACHINES_WATCH, () => {});
        const inGrace = engine.open('machines.list');
        back();
        await clock.advance(501);
        const left = [engine.open('machines.list'), store.reading(MACHINES_KEY).kind];
        await clock.advance(KEPT_FOR);

        // Then - the watch stayed open across the flap, closed after the grace with its answer kept, then forgotten
        expect(inGrace).toBe(1);
        expect(left).toStrictEqual([0, 'read']);
        expect(store.reading(MACHINES_KEY)).toStrictEqual({ kind: 'reading' });
    });

    test('a read held again after its grace shows its kept answer at once, and is asked again quietly', async () => {
        // Given - engine.read answered, then left past its grace, and the engine restarted meanwhile
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { stage, store } = staged();
        const leave = store.subscribe(engineRead(), () => {});
        await clock.advance(1);
        leave();
        await clock.advance(501);
        stage.restart({ ...HELLO, pid: 2 });
        await clock.advance(1);
        const whileKept = asked(stage, 'engine.read');

        // When - a page holds it again
        let told = 0;
        store.subscribe(engineRead(), () => {
            told += 1;
        });
        const atOnce = store.reading(READ_KEY);
        await clock.advance(1);

        // Then - the restart asked nothing kept, the answer showed at once, and its equal re-ask told no one
        expect(whileKept).toBe(1);
        expect(atOnce).toStrictEqual({
            at: new Date('2026-11-13T10:00:00.001Z'),
            done: true,
            kind: 'read',
            out: { State: 'running' },
        });
        expect(asked(stage, 'engine.read')).toBe(2);
        expect(told).toBe(0);
    });

    test('the answers nobody holds are kept two at most, the one left first forgotten first', async () => {
        // Given - three requests answered, each held by one subscriber
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { store } = staged();
        const leaves = [
            engineRead(),
            MACHINES_WATCH,
            { ...engineRead(), target: { name: 'api', on: 'mac' } },
        ].map((one) => store.subscribe(one, () => {}));
        await clock.advance(1);

        // When - each is left in turn, and their graces pass
        for (const leave of leaves) {
            leave();
        }
        await clock.advance(501);

        // Then - the first left is forgotten, the two others are kept
        expect(
            [READ_KEY, MACHINES_KEY, API_KEY].map((key) => store.reading(key).kind),
        ).toStrictEqual(['reading', 'read', 'read']);
    });

    test('an engine that comes back with another hello is asked every read again', async () => {
        // Given - engine.read answered by the first engine
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { stage, store } = staged();
        store.subscribe(engineRead(), () => {});
        await clock.advance(1);

        // When - the engine restarts: a new pid, a new start, and another answer
        stage.restart({ ...HELLO, pid: 2, started_at: '2026-11-13T10:00:05Z' }, RESTARTED);
        await clock.advance(1);

        // Then - the read was asked twice and shows what the new engine says
        expect(asked(stage, 'engine.read')).toBe(2);
        expect(store.reading('call engine.read {} {}')).toStrictEqual({
            at: new Date('2026-11-13T10:00:00.001Z'),
            done: true,
            kind: 'read',
            out: { State: 'running', Version: 'v2' },
        });
    });

    test('the same hello seen again asks nothing, and a link that flips asks again on its return', async () => {
        // Given - engine.read answered
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { stage, store } = staged();
        store.subscribe(engineRead(), () => {});
        await clock.advance(1);

        // When - the same hello is shown again, then the link goes absent and comes back
        stage.show({ hello: HELLO, kind: 'open', refused: undefined, socket: 'replay of test' });
        await clock.advance(1);
        const sameEngine = asked(stage, 'engine.read');
        stage.show({ kind: 'absent', reason: 'connection refused', retryIn: 1000, socket: 's' });
        await clock.advance(1);
        const whileAbsent = store.reading('call engine.read {} {}').kind;
        stage.show({ hello: HELLO, kind: 'open', refused: undefined, socket: 'replay of test' });
        await clock.advance(1);

        // Then - one ask for the same engine, the answer kept while absent, one more on the return
        expect(sameEngine).toBe(1);
        expect(whileAbsent).toBe('read');
        expect(asked(stage, 'engine.read')).toBe(2);
    });

    test('an answer that comes after a newer one is discarded', async () => {
        // Given - a replay whose first engine.read answers late, the next at once
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { client, store } = staged({
            gestureFrame: 0,
            latency: (request, before) =>
                'call' in request && request.call === 'engine.read' && before === 0 ? 100 : 0,
        });
        store.subscribe(engineRead(), () => {});
        await clock.advance(1);

        // When - the engine is stopped and engine.read asked again before its first answer arrives
        const stopped = client.call('engine.stop', { Confirm: 'yes' });
        await clock.advance(1);
        await stopped;
        store.reread(['engine.read']);
        await clock.advance(1);
        const newer = store.reading('call engine.read {} {}');
        await clock.advance(200);

        // Then - the late answer of the running engine never replaced the newer one
        expect(newer).toStrictEqual({
            at: new Date('2026-11-13T10:00:00.002Z'),
            done: true,
            kind: 'read',
            out: { State: 'stopped' },
        });
        expect(store.reading('call engine.read {} {}')).toStrictEqual(newer);
    });

    test('each request marks its first out and its end, one measure per name however often it is asked', async () => {
        // Given - a store over a replay, machines.list watched and api's engine.read read
        performance.clearMeasures();
        const { stage, store } = staged();
        const watch: Asked = { input: {}, kind: 'watch', name: 'machines.list', target: undefined };
        store.subscribe(watch, () => {});
        store.subscribe({ ...engineRead(), target: { name: 'api', on: 'mac' } }, () => {});

        // When - both are answered, then asked again on a gesture's reread
        await waitUntil(() => store.reading(API_KEY).kind === 'read', { why: 'the read answered' });
        store.reread(['engine.read', 'machines.list']);
        await waitUntil(() => stage.open() === 0, { why: 'the reread answered' });

        // Then - one measure per name: the read's first out and end, the watch, which has not ended, its first out
        const names = performance
            .getEntriesByType('measure')
            .map((measure) => measure.name)
            .filter((name) => name.startsWith('read '));
        expect(names.toSorted()).toStrictEqual([
            `read done ${API_KEY}`,
            `read first ${API_KEY}`,
            'read first watch machines.list {} {}',
        ]);
    });

    test('an engine whose stream says it does not fit is incompatible and asked nothing more, until another fits', async () => {
        // Given - machines.list settled over an engine whose state stream opens newer than this window
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();
        engine.answer('engine.state', {
            out: { event: 'opened', fit: { app: 1, engine: 2, verdict: 'app-older' }, rev: 1 },
        });
        const unfit = store.link();

        // When - a gesture rereads it, then the engine restarts as another whose stream fits
        store.reread(['machines.list']);
        const whileUnfit = engine.asked('machines.list');
        engine.restart();
        await clock.advance(1);
        engine.answer('engine.state', {
            out: { event: 'opened', fit: { app: 1, engine: 1, verdict: 'fits' }, rev: 1 },
        });

        // Then - the verdict is the engine's, nothing was asked of it, and the fitting engine is open and asked again
        expect(unfit).toMatchObject({ fit: 'app-older', kind: 'incompatible', socket: 'fake' });
        expect(whileUnfit).toBe(1);
        expect(store.link()?.kind).toBe('open');
        expect(engine.asked('machines.list')).toBe(2);
        expect(engine.asked('engine.state')).toBe(2);
    });
});
