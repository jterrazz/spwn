import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import type { Asked } from '../entry';
import { DONE, EVERY, KEY, PENDING, RUNNING, settled, STOPPED } from './fake-engine';

const ENGINE_READ: Asked = { input: {}, kind: 'read', name: 'engine.read', target: undefined };

const UNREACHABLE = {
    code: 'remote.unreachable',
    message: 'studio did not answer',
    type: 'unreachable',
};

function settledOn(at: Date | undefined, out: unknown) {
    return { at, done: true, kind: 'read', out };
}

describe('ask', () => {
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
        expect(store.reading(KEY)).toStrictEqual({
            at: new Date('2026-11-13T10:01:00.001Z'),
            done: true,
            kind: 'read',
            out: RUNNING.out,
        });
    });

    test('a dropped watch keeps its answer unmarked while it reconnects, a dropped read keeps it marked stale', async () => {
        // Given - machines.list settled on running, and engine.read answered beside it
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();
        store.subscribe(ENGINE_READ, () => {});
        await clock.advance(1);
        engine.answer('engine.read', RUNNING);
        const shown = store.reading(KEY);

        // When - the cadence asks both again and the connection of each ends unanswered
        await clock.advance(EVERY);
        engine.end('machines.list');
        engine.end('engine.read');
        await clock.advance(1);

        // Then - the watch's reading is the one shown, the read's answer carries a stale mark of no engine word
        const read = store.reading('call engine.read {} {}');
        expect(store.reading(KEY)).toBe(shown);
        expect(read.kind === 'read' ? [read.out, read.stale?.wire] : []).toStrictEqual([
            RUNNING.out,
            undefined,
        ]);
    });

    test('a refusal of the subject itself replaces the shown answer', async () => {
        // Given - machines.list settled on running
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();

        // When - the next re-ask is answered that the machine is not found
        await clock.advance(EVERY);
        engine.answer('machines.list', {
            error: { code: 'unknown_vm', message: 'no such machine', type: 'not-found' },
        });

        // Then - the refusal is shown
        expect(store.reading(KEY).kind).toBe('failed');
    });

    test("a gesture's reread and an engine restart hold a watch's pending outs over its settled answer", async () => {
        // Given - two stores, each with machines.list settled on running
        using _ = clock.run('2026-11-13T10:00:00Z');
        const gesture = await settled();
        const restart = await settled();

        // When - one is reread after a gesture, the other's engine comes back as another, and each answers pending first
        gesture.store.reread(['machines.list']);
        restart.engine.restart();
        gesture.engine.answer('machines.list', PENDING);
        restart.engine.answer('machines.list', PENDING);
        const meanwhile = [gesture.store.reading(KEY), restart.store.reading(KEY)];
        gesture.engine.answer('machines.list', STOPPED, DONE);
        restart.engine.answer('machines.list', STOPPED, DONE);

        // Then - both were asked again, the pending value never shown, and each shows the last
        const [first, second] = ['2026-11-13T10:00:00.001Z', '2026-11-13T10:00:00.002Z'].map(
            (at) => new Date(at),
        );
        expect([
            gesture.engine.asked('machines.list'),
            restart.engine.asked('machines.list'),
        ]).toStrictEqual([2, 2]);
        expect(meanwhile).toStrictEqual([
            settledOn(first, RUNNING.out),
            settledOn(second, RUNNING.out),
        ]);
        expect([gesture.store.reading(KEY), restart.store.reading(KEY)]).toStrictEqual([
            settledOn(second, STOPPED.out),
            settledOn(second, STOPPED.out),
        ]);
    });
});
