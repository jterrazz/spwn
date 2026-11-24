import { describe, expect, test } from 'vitest';

import { Client } from '../../engine-link/client';
import { stagedReplay } from '../replay';
import type { Recording } from '../replay';

const LISTED = { call: 'engine.read', in: {} };
const RELEASE = { call: 'engine.read-release', in: {} };

const answer = (State: string) => [{ out: { State }, v: 2 }];

const RECORDING: Recording = {
    exchanges: [
        { answers: answer('running'), request: LISTED },
        { answers: answer('checked'), request: RELEASE },
    ],
    gestures: [
        {
            after: [{ answers: answer('stopped'), request: LISTED }],
            answers: [
                { progress: { task: 'Stopping the engine' }, v: 2 },
                { progress: { task: 'Stopping the engine', done: true }, v: 2 },
                { out: { Stopped: true }, v: 2 },
            ],
            during: [
                { answers: answer('stopping'), request: LISTED },
                { answers: answer('unchecked'), request: RELEASE },
            ],
            request: { call: 'engine.stop', in: { Confirm: 'yes' } },
        },
    ],
    station: 'dev',
    world: 'test',
};

describe('played', () => {
    test('a gesture recorded with its reads during it answers them so from its first frame, then as after it', async () => {
        // Given - a replay of a world whose stop was recorded with the reads it answered while it ran
        const client = new Client(stagedReplay(RECORDING).transport);
        const during: unknown[] = [];

        // When - the stop is sent, the reads asked at its first progress, then again once it is done
        await client.call(
            'engine.stop',
            { Confirm: 'yes' },
            {
                progress: () => {
                    if (during.length === 0) {
                        during.push(
                            client.call('engine.read', {}),
                            client.call('engine.read-release', {}),
                        );
                    }
                },
            },
        );
        const asked = await Promise.all(during);
        const after = [
            await client.call('engine.read', {}),
            await client.call('engine.read-release', {}),
        ];

        // Then - while it ran, both as recorded during it; once done, the read it changed as after it, the other as before
        expect(asked).toStrictEqual([{ State: 'stopping' }, { State: 'unchecked' }]);
        expect(after).toStrictEqual([{ State: 'stopped' }, { State: 'checked' }]);
    });

    test('a gesture of the held call stops after its first frame, and its reads answer as while it ran', async () => {
        // Given - the same replay, the stop's call held
        const stage = stagedReplay(RECORDING, undefined, { gestureFrame: 0, held: 'engine.stop' });
        const client = new Client(stage.transport);
        const tasks: string[] = [];

        // When - the stop is sent, and the engine read once its first progress comes
        const first = Promise.withResolvers<unknown>();
        const stop = client.call(
            'engine.stop',
            { Confirm: 'yes' },
            {
                progress: (progress) => {
                    tasks.push(progress.task);
                    first.resolve(client.call('engine.read', {}));
                },
            },
        );
        const read = await first.promise;

        // Then - one frame, the stop still open, and the read as recorded during it
        expect(tasks).toStrictEqual(['Stopping the engine']);
        expect(read).toStrictEqual({ State: 'stopping' });
        expect(stage.open()).toBe(1);

        // And - the engine closing the connection ends the held stop as lost, so no timer outlives the test
        stage.drop();
        await expect(stop).rejects.toMatchObject({ fate: 'lost', wire: undefined });
        expect(stage.open()).toBe(0);
    });
});
