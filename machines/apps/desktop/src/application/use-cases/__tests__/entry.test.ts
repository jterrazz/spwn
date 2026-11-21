import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { DONE, EVERY, KEY, rowsOf, settled } from './fake-engine';

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
});
