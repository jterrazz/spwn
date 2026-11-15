import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../client';
import type { Asked } from '../entry';
import { stagedReplay } from '../replay';
import type { Pace, Recording } from '../replay';
import { ReadStore } from '../store';

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

const HELLO = { pid: 1, started_at: '2026-11-13T09:00:00Z', version: 'v1' };

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
    return { client, stage, store: new ReadStore(client, { grace: 500 }) };
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
            done: true,
            kind: 'read',
            out: { State: 'running', Version: 'api' },
        });
    });

    test('the last subscriber leaving closes the watch after its grace, and one back within it keeps it', async () => {
        // Given - a watch of machines.list held by one subscriber
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { store } = staged();
        const watch: Asked = { input: {}, kind: 'watch', name: 'machines.list', target: undefined };
        const leave = store.subscribe(watch, () => {});
        await clock.advance(1);
        const watching = store.reading('watch machines.list {} {}').kind;

        // When - it leaves and comes back within the grace, then leaves for longer than it
        leave();
        await clock.advance(400);
        const back = store.subscribe(watch, () => {});
        const kept = store.reading('watch machines.list {} {}').kind;
        back();
        await clock.advance(501);

        // Then - the answer was kept across the flap, and dropped once the grace was spent
        expect([watching, kept]).toStrictEqual(['read', 'read']);
        expect(store.reading('watch machines.list {} {}')).toStrictEqual({ kind: 'reading' });
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
        expect(newer).toStrictEqual({ done: true, kind: 'read', out: { State: 'stopped' } });
        expect(store.reading('call engine.read {} {}')).toStrictEqual(newer);
    });
});
