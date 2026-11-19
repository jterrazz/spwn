import { describe, expect, test } from 'vitest';

import { helloOf } from '../hello';
import type { Recording } from '../replay';

const READ = { call: 'engine.read', in: {} };

const RECORDING: Recording = {
    exchanges: [
        { answers: [{ out: { State: 'running' }, v: 2 }], request: READ },
        {
            answers: [{ out: { State: 'running' }, v: 2 }],
            request: { ...READ, target: 'mac-mini' },
        },
    ],
    gestures: [
        {
            after: [],
            answers: [{ out: { Stopped: true }, v: 2 }],
            request: { call: 'engine.stop', in: { Confirm: 'yes' } },
        },
    ],
    station: 'dev',
    world: 'test',
};

describe('hello', () => {
    test("a world's hello offers the calls it holds answers for, its reads and its gestures, each once", () => {
        // Given - a world holding one read twice and one gesture

        // When - its hello is made
        const hello = helloOf(RECORDING);

        // Then - both calls, sorted, under the world's station
        expect(hello).toStrictEqual({
            calls: ['engine.read', 'engine.stop'],
            pid: 0,
            started_at: '',
            version: 'dev',
        });
    });
});
