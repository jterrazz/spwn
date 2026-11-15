import { describe, expect, test } from 'vitest';

import { reachOf } from '../reach';

describe('reach', () => {
    test('a socket nobody answers on is said as such, never as an engine stopped', () => {
        // Given - the link of a window whose socket refuses, one that is open, and none heard yet
        const absent = {
            kind: 'absent',
            reason: 'connection refused',
            retryIn: 4000,
            socket: '/run/engine.sock',
        } as const;
        const open = {
            hello: undefined,
            kind: 'open',
            refused: undefined,
            socket: '/run/engine.sock',
        } as const;

        // When - each is read as the transport's reach
        const reaches = [reachOf(absent), reachOf(open), reachOf(undefined)];

        // Then - the absent one says no engine answers there, and none speaks for the engine's own state
        expect(reaches).toStrictEqual([
            {
                kind: 'unanswered',
                reason: 'connection refused',
                retryIn: 4000,
                said: 'No engine answers at /run/engine.sock',
                socket: '/run/engine.sock',
            },
            { kind: 'answering', socket: '/run/engine.sock' },
            { kind: 'connecting', socket: undefined },
        ]);
    });
});
