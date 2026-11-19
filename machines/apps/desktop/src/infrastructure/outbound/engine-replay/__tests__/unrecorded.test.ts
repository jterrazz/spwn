import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { Client } from '../../engine-link/client';
import { replayTransport } from '../replay';
import type { Recording } from '../replay';
import { UnrecordedGestureError } from '../unrecorded';

const READ = { call: 'engine.read', in: {} };

const RECORDING: Recording = {
    exchanges: [{ answers: [{ out: { State: 'running' }, v: 2 }], request: READ }],
    gestures: [
        {
            after: [{ answers: [{ out: { State: 'stopped' }, v: 2 }], request: READ }],
            answers: [{ out: { Stopped: true }, v: 2 }],
            request: { call: 'engine.stop', in: { Confirm: 'yes' } },
        },
        {
            after: [],
            answers: [{ out: { Stopped: true }, v: 2 }],
            request: { call: 'engine.stop', in: { Confirm: 'yes' }, target: { name: 'mac-mini' } },
        },
    ],
    station: 'dev',
    world: 'test',
};

const SAID =
    'engine.stop was sent with {"in":{}}, but is recorded only with {"in":{"Confirm":"yes"}} or {"in":{"Confirm":"yes"},"target":{"name":"mac-mini"}}';

describe('unrecorded', () => {
    test('a gesture sent with an input no recording holds is thrown and refused, naming what was sent and what is recorded', async () => {
        // Given - a replay whose stop is recorded with its confirmation, here and on mac-mini
        using _ = clock.run('2026-11-19T10:00:00Z');
        const client = new Client(replayTransport(RECORDING));

        // When - the stop is sent with no confirmation
        const [refused, advanced] = await Promise.allSettled([
            client.call('engine.stop', {}),
            clock.advance(0),
        ]);

        // Then - the replay throws at once, the call ends on the same words, and the engine still reads running
        expect(advanced).toStrictEqual({
            reason: new UnrecordedGestureError(SAID),
            status: 'rejected',
        });
        expect(refused).toMatchObject({
            reason: { message: SAID, wire: undefined },
            status: 'rejected',
        });
        const read = client.call('engine.read', {});
        await clock.advance(0);
        await expect(read).resolves.toStrictEqual({ State: 'running' });
    });

    test('a gesture of a call no recording holds stays no answer, as a read does', async () => {
        // Given - the same replay
        const client = new Client(replayTransport(RECORDING));

        // When - a start, which the world never recorded, is sent
        const unanswered = client.call('engine.start', {});

        // Then - no answer, and nothing said
        await expect(unanswered).rejects.toMatchObject({ message: '', wire: undefined });
    });
});
