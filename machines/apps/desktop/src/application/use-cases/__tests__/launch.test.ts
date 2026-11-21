import { describe, expect, test } from 'vitest';

import { EngineError } from '../../../domain/engine-error';
import type { LaunchPort } from '../../ports/launch.port';
import { holdsData, Launch, LAUNCH_CAP } from '../launch';

type Due = { at: number; run: () => void };

function fakeShell(since = 0) {
    const dues: Due[] = [];
    let now = since;
    let shows = 0;
    let loaded: (() => void) | undefined;
    const held = new Promise<void>((resolve) => {
        loaded = resolve;
    });
    const port: LaunchPort = {
        after: (ms, run) => {
            dues.push({ at: now + ms, run });
        },
        faces: async () => {
            await held;
        },
        show: async () => {
            shows += 1;
            await Promise.resolve();
        },
        since: () => now,
    };
    return {
        advance: (to: number) => {
            now = to;
            for (const due of dues.filter((one) => one.at <= to)) {
                dues.splice(dues.indexOf(due), 1);
                due.run();
            }
        },
        load: async () => {
            loaded?.();
            await new Promise<void>((resolve) => {
                setImmediate(resolve);
            });
        },
        port,
        shows: () => shows,
    };
}

describe('launch', () => {
    test('the window shows once the frame holds data and its faces are loaded', async () => {
        // Given - a launch begun, its faces loaded at 120 ms
        const shell = fakeShell();
        const launch = new Launch(shell.port);
        launch.begin();
        shell.advance(120);
        await shell.load();

        // When - the frame draws its first data at 300 ms, then the cap passes
        const before = shell.shows();
        shell.advance(300);
        launch.drawn();
        shell.advance(LAUNCH_CAP);
        launch.drawn();

        // Then - shown once, at the data
        expect(before).toBe(0);
        expect(shell.shows()).toBe(1);
    });

    test('data drawn before the faces load waits for them', async () => {
        // Given - a launch whose frame holds data at 200 ms
        const shell = fakeShell();
        const launch = new Launch(shell.port);
        launch.begin();
        shell.advance(200);
        launch.drawn();
        const before = shell.shows();

        // When - the faces load at 250 ms
        shell.advance(250);
        await shell.load();

        // Then - shown at the faces, not before
        expect(before).toBe(0);
        expect(shell.shows()).toBe(1);
    });

    test('at the cap the window shows whatever is drawn, and only once', async () => {
        // Given - a launch with no data and no faces
        const shell = fakeShell();
        const launch = new Launch(shell.port);
        launch.begin();

        // When - the cap passes, then the data and the faces come
        shell.advance(LAUNCH_CAP - 1);
        const before = shell.shows();
        shell.advance(LAUNCH_CAP);
        launch.drawn();
        await shell.load();

        // Then - one show, at 800 ms
        expect(before).toBe(0);
        expect(shell.shows()).toBe(1);
    });

    test('the cap counts from the page load, not from the script', () => {
        // Given - a script that begins 500 ms into the page load
        const shell = fakeShell(500);
        const launch = new Launch(shell.port);

        // When - it begins, and the clock reaches 800 ms
        launch.begin();
        shell.advance(799);
        const before = shell.shows();
        shell.advance(800);

        // Then - shown at 800 ms of the page
        expect(before).toBe(0);
        expect(shell.shows()).toBe(1);
    });

    test('a script that begins past the cap shows at once', () => {
        // Given - a script that begins 1.2 s into the page load
        const shell = fakeShell(1200);
        const launch = new Launch(shell.port);

        // When - it begins
        launch.begin();
        shell.advance(1200);

        // Then - shown with no wait
        expect(shell.shows()).toBe(1);
    });

    test('the frame holds data at its first out, its refusal, or no engine', () => {
        // Given - the link and machines.list in each state of a launch
        const answering = { kind: 'answering', socket: 's' } as const;
        const connecting = { kind: 'connecting', socket: undefined } as const;
        const unanswered = {
            kind: 'unanswered',
            reason: 'connection refused',
            retryIn: 30_000,
            said: 'No engine answers at s',
            socket: 's',
        } as const;
        const reading = { kind: 'reading' } as const;
        const partial = { done: false, kind: 'read', out: {} } as const;
        const refused = { error: new EngineError(undefined, 'refused'), kind: 'failed' } as const;

        // When - each pair is judged
        const judged = [
            holdsData(connecting, reading),
            holdsData(answering, reading),
            holdsData(answering, partial),
            holdsData(answering, refused),
            holdsData(unanswered, reading),
        ];

        // Then - only a link still connecting or a list still unanswered holds none
        expect(judged).toStrictEqual([false, false, true, true, true]);
    });
});
