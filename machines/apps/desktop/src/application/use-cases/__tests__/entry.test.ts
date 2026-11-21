import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { DONE, EVERY, KEY, rowsOf, settled } from './fake-engine';

const UNREACHABLE = {
    code: 'remote.unreachable',
    message: 'studio did not answer',
    type: 'unreachable',
};
const NOT_FOUND = { code: 'unknown_vm', message: 'no such machine', type: 'not-found' };

describe('entry', () => {
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

    test('an equal answer keeps the reading and its instant, and a stale mark carries the instant of the last answer heard', async () => {
        // Given - machines.list settled on running, then answered equal on the next cadence
        using _ = clock.run('2026-11-13T10:00:00Z');
        const { engine, store } = await settled();
        const shown = store.reading(KEY);
        await clock.advance(EVERY);
        engine.answer('machines.list', { out: { State: 'running' } }, DONE);
        const equal = store.reading(KEY);

        // When - the cadence after is refused as a far Mac that missed the ask
        await clock.advance(EVERY);
        engine.answer('machines.list', { error: UNREACHABLE });
        const stale = store.reading(KEY);

        // Then - the equal answer was the reading shown, as of its first answer; the stale one is as of the equal one
        expect(equal).toBe(shown);
        expect(
            [shown, stale].map((one) => (one.kind === 'read' ? one.at : undefined)),
        ).toStrictEqual([
            new Date('2026-11-13T10:00:00.001Z'),
            new Date('2026-11-13T10:00:30.001Z'),
        ]);
    });

    test("an answer's instant is read off the store's clock", async () => {
        // Given - a store whose clock says eight o'clock
        using _ = clock.run('2026-11-13T10:00:00Z');

        // When - machines.list is answered
        const { store } = await settled(undefined, { now: () => new Date('2026-11-13T08:00:00Z') });

        // Then - the reading is as of the store's clock
        const reading = store.reading(KEY);
        expect(reading.kind === 'read' ? reading.at : undefined).toStrictEqual(
            new Date('2026-11-13T08:00:00Z'),
        );
    });

    test('a refusal said again in the same words keeps the reading itself, stale or failed', async () => {
        // Given - one store whose answer a far Mac missed, another whose machine was not found
        using _ = clock.run('2026-11-13T10:00:00Z');
        const missed = await settled();
        const lost = await settled();
        const refuse = async (error: object) => {
            await clock.advance(EVERY);
            missed.engine.answer('machines.list', { error: UNREACHABLE });
            lost.engine.answer('machines.list', { error });
        };
        await refuse(NOT_FOUND);
        const shown = [missed.store.reading(KEY), lost.store.reading(KEY)];

        // When - the next cadence is refused again, each in the same words as new objects
        await refuse({ ...NOT_FOUND });

        // Then - each reading is the one shown
        expect(shown.map((reading) => reading.kind)).toStrictEqual(['read', 'failed']);
        expect(missed.store.reading(KEY)).toBe(shown[0]);
        expect(lost.store.reading(KEY)).toBe(shown[1]);
    });
});
