import type { Request } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { refusalOf, stagedOf } from '../staged';

const PORTS: Request = { call: 'ports.list', id: 'one', in: { Of: 'macbook-pro' }, v: 2 };
const AWAKE: Request = { id: 'two', in: {}, v: 2, watch: 'awake.read' };
const REFUSAL = { error: { message: 'macbook-air does not answer', type: 'unreachable' }, v: 2 };

describe('staged', () => {
    test("the world's first unreachable refusal is the one a stale read replays", () => {
        // Given - a world whose first exchange answers and whose second is refused as unreachable
        const recording = {
            exchanges: [
                {
                    answers: [
                        { out: {}, v: 2 },
                        { done: true, v: 2 },
                    ],
                },
                { answers: [REFUSAL] },
            ],
        };

        // When - its refusal is looked for
        const found = refusalOf(recording);

        // Then - it is the refused frame, whole
        expect(found).toStrictEqual(REFUSAL);
    });

    test('an unread name is held open with no frame, whatever it is', () => {
        // Given - a pace that leaves awake.read unread
        const staged = { unread: 'awake.read' };

        // When - a watch of it and a call of another name are staged
        const plays = [stagedOf(AWAKE, 0, staged, REFUSAL), stagedOf(PORTS, 0, staged, REFUSAL)];

        // Then - the watch stays open with nothing played, the call plays as recorded
        expect(plays).toStrictEqual([{ frames: [], gap: 0, held: true }, undefined]);
    });

    test('a stale name answers as recorded the first time, then is refused', () => {
        // Given - a pace that makes ports.list stale
        const staged = { stale: 'ports.list' };

        // When - it is staged as first asked, then as asked again
        const plays = [stagedOf(PORTS, 0, staged, REFUSAL), stagedOf(PORTS, 1, staged, REFUSAL)];

        // Then - the first plays as recorded, the second is the refusal alone
        expect(plays).toStrictEqual([undefined, { frames: [REFUSAL], gap: 0 }]);
    });
});
