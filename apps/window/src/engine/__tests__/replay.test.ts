import { describe, expect, test } from 'vitest';

import { Client } from '../client';
import { replayTransport } from '../replay';
import type { Recording } from '../replay';

const LISTED = {
    request: { call: 'engine.read', in: {} },
    answers: [{ out: { State: 'running' }, v: 2 }],
};

const RECORDING: Recording = {
    exchanges: [LISTED],
    gestures: [
        {
            after: [{ answers: [{ out: { State: 'stopped' }, v: 2 }], request: LISTED.request }],
            answers: [
                { progress: { task: 'Stopping the engine' }, v: 2 },
                { out: { Stopped: true }, v: 2 },
            ],
            request: { call: 'engine.stop', in: { Confirm: 'yes' } },
        },
    ],
    station: 'dev',
    world: 'test',
};

describe('replay', () => {
    test('a recorded gesture plays its frames, then the reads answer as they did after it', async () => {
        // Given - a replay of a world where the engine runs
        const client = new Client(replayTransport(RECORDING));
        const before = await client.call('engine.read', {});
        const tasks: string[] = [];

        // When - the recorded stop is sent, then the engine read again
        const out = await client.call(
            'engine.stop',
            { Confirm: 'yes' },
            {
                progress: (progress) => {
                    tasks.push(progress.task);
                },
            },
        );
        const after = await client.call('engine.read', {});

        // Then - its progress and out as recorded, and the read as it was after it
        expect(before).toStrictEqual({ State: 'running' });
        expect(tasks).toStrictEqual(['Stopping the engine']);
        expect(out).toStrictEqual({ Stopped: true });
        expect(after).toStrictEqual({ State: 'stopped' });
    });

    test('a gesture not recorded is no answer, and changes nothing', async () => {
        // Given - the same replay
        const client = new Client(replayTransport(RECORDING));

        // When - a stop with another confirmation is sent
        const refused = client.call('engine.stop', {});

        // Then - the world holds no answer to it, and the engine still reads running
        await expect(refused).rejects.toThrow('test holds no answer to call engine.stop');
        await expect(client.call('engine.read', {})).resolves.toStrictEqual({ State: 'running' });
    });
});
