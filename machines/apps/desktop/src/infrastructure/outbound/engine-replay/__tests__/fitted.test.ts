import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { fittedOf } from '../fitted';
import { stagedReplay } from '../replay';
import type { Recording } from '../replay';

const RECORDING: Recording = { exchanges: [], station: 'dev', world: 'test' };

const STATE = {
    id: '1',
    in: { keys: ['local'], revision: 1 },
    v: 2,
    watch: 'engine.state',
} as const;

function asked(stage: ReturnType<typeof stagedReplay>) {
    const frames: unknown[] = [];
    const ends: string[] = [];
    stage.transport.open(STATE, {
        end: (reason) => {
            ends.push(reason);
        },
        frame: (frame) => {
            frames.push(frame);
        },
    });
    return { ends, frames };
}

describe('fitted', () => {
    test('each fit the query names is a verdict, and any other leaves the world fitting', () => {
        // Given - the two fits a proof build plays, a name it does not know, and none

        // When - each is read
        const fits = ['engine-older', 'app-older', 'stale', null].map((fit) => fittedOf(fit));

        // Then - an older engine and an older window, then none twice
        expect(fits).toStrictEqual(['engine-older', 'app-older', undefined, undefined]);
    });

    test("an unfit world's engine opens its state stream with that verdict and ends it, until it restarts", async () => {
        // Given - a world whose engine is older than the window
        using _ = clock.run('2026-11-21T09:00:00Z');
        const stage = stagedReplay(RECORDING, undefined, { fit: 'engine-older', gestureFrame: 0 });

        // When - the window asks its state stream, then the engine restarts and is asked again
        const before = asked(stage);
        await clock.advance(1);
        stage.restart({ pid: 2, started_at: 'restarted', version: 'dev' });
        const after = asked(stage);
        await clock.advance(1);

        // Then - an opening alone, below the window's revision, ended; then the opening that fits and its snapshot, held
        expect(before.frames).toMatchObject([
            { out: { event: 'opened', fit: { app: 1, engine: 0, verdict: 'engine-older' } } },
        ]);
        expect(before.ends).toStrictEqual(['answered']);
        expect(after.frames).toMatchObject([
            { out: { event: 'opened', fit: { app: 1, engine: 1, verdict: 'fits' } } },
            { out: { event: 'snapshot' } },
        ]);
        expect(after.ends).toStrictEqual([]);
    });
});
