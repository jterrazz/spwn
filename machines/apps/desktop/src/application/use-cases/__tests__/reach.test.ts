import { describe, expect, test } from 'vitest';

import { PAGE_REVISION } from '../../../domain/fit';
import { judged, reachOf } from '../reach';

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

    test('an engine answering another revision is judged incompatible and running, never absent', () => {
        // Given - hellos naming no revision, one below the revision drawn, that revision, and one above it
        const socket = '/run/engine.sock';
        const unnamed = { executable: '/u/bin/station', pid: 1, started_at: '', version: 'v1' };
        const hellos = [
            unnamed,
            ...[PAGE_REVISION - 1, PAGE_REVISION, PAGE_REVISION + 1].map((page) => ({
                ...unnamed,
                page,
            })),
        ];

        // When - each opens the link, and is judged and read as the reach
        const reaches = hellos.map((hello) =>
            reachOf(judged({ hello, kind: 'open', refused: undefined, socket })),
        );

        // Then - the older engine twice, the fitting one answering, the older window, each with the hello's station
        const unfit = { executable: '/u/bin/station', kind: 'incompatible', socket };
        expect(reaches).toStrictEqual([
            { ...unfit, fit: 'engine-older' },
            { ...unfit, fit: 'engine-older' },
            { kind: 'answering', socket },
            { ...unfit, fit: 'app-older' },
        ]);
    });

    test('a link that does not open is passed as it came, judged of nothing', () => {
        // Given - a link connecting and one absent
        const links = [
            { kind: 'connecting', socket: '/run/engine.sock' },
            { kind: 'absent', reason: 'refused', retryIn: 1000, socket: '/run/engine.sock' },
        ] as const;

        // When - each is judged
        const judgedLinks = links.map((link) => judged(link));

        // Then - each is the same link
        expect(judgedLinks).toStrictEqual(links);
    });
});
