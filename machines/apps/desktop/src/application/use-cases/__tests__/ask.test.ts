import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { DONE, EVERY, KEY, PENDING, RUNNING, settled, STOPPED } from './fake-engine';

const UNREACHABLE = {
    code: 'remote.unreachable',
    message: 'studio did not answer',
    type: 'unreachable',
};

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
        const shown = { done: true, kind: 'read', out: RUNNING.out };
        const last = { done: true, kind: 'read', out: STOPPED.out };
        expect([
            gesture.engine.asked('machines.list'),
            restart.engine.asked('machines.list'),
        ]).toStrictEqual([2, 2]);
        expect(meanwhile).toStrictEqual([shown, shown]);
        expect([gesture.store.reading(KEY), restart.store.reading(KEY)]).toStrictEqual([
            last,
            last,
        ]);
    });
});
