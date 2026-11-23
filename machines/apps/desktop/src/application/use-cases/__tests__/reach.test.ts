import { describe, expect, test } from 'vitest';

import { judgedOf, reachOf } from '../reach';

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

    test('an engine is incompatible on its own verdict alone, running and never absent', () => {
        // Given - one engine's open link, and the verdicts it may have sent: none yet, fits, and the two misfits
        const socket = '/run/engine.sock';
        const hello = { executable: '/u/bin/station', pid: 1, started_at: '', version: 'v1' };
        const link = { hello, kind: 'open', refused: undefined, socket } as const;

        // When - the link is judged by each, and read as the reach
        const reaches = ([undefined, 'fits', 'engine-older', 'app-older'] as const).map((fit) =>
            reachOf(judgedOf(link, fit)),
        );

        // Then - answering until a misfit, then incompatible with that verdict and the hello's station
        const unfit = { executable: '/u/bin/station', kind: 'incompatible', socket };
        expect(reaches).toStrictEqual([
            { kind: 'answering', socket },
            { kind: 'answering', socket },
            { ...unfit, fit: 'engine-older' },
            { ...unfit, fit: 'app-older' },
        ]);
    });

    test('a link that does not open is passed as it came, whatever the verdict', () => {
        // Given - a link connecting and one absent
        const links = [
            { kind: 'connecting', socket: '/run/engine.sock' },
            { kind: 'absent', reason: 'refused', retryIn: 1000, socket: '/run/engine.sock' },
        ] as const;

        // When - each is judged by an older engine's verdict
        const judgedLinks = links.map((link) => judgedOf(link, 'engine-older'));

        // Then - each is the same link
        expect(judgedLinks).toStrictEqual(links);
    });
});
