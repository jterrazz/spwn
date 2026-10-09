import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { gate } from './server.fixtures.ts';

const MINUTE = 60 * 1000;

function fetches(from: number, to: number): string {
    return `(async () => {
        for (let call = ${from}; call <= ${to}; call += 1) {
            await fetch('/data.json?call=' + call);
        }
        await fetch('/note.txt');
        await fetch('/bare');
    })()`;
}

function ends(urls: string[]) {
    return { count: urls.length, first: urls[0], last: urls.at(-1) };
}

describe('Session', () => {
    test('expires five minutes after its last use, and thirty after it opened whatever its use', async () => {
        // Given - a session opened at 08:00, then used at 08:04 and 08:28
        using _ = clock.at('2026-10-09T08:00:00Z');
        await using sidecar = await gate();
        const { session } = await sidecar.sessions.create('site');
        const opened = Date.parse('2026-10-09T08:00:00Z');
        const fresh = [
            session.expired(opened + 5 * MINUTE),
            session.expired(opened + 5 * MINUTE + 1),
        ];
        await clock.advance(4 * MINUTE);
        session.touch();
        const used = [
            session.expired(opened + 9 * MINUTE),
            session.expired(opened + 9 * MINUTE + 1),
        ];
        await clock.advance(24 * MINUTE);
        session.touch();

        // When - its expiry is read just before and just after thirty minutes
        const late = [
            session.expired(opened + 30 * MINUTE),
            session.expired(opened + 30 * MINUTE + 1),
        ];

        // Then - idle past five minutes it expires, and at thirty minutes it expires though used
        expect({ fresh, late, used }).toStrictEqual({
            fresh: [false, true],
            late: [false, true],
            used: [false, true],
        });
    });

    test('keeps the last five hundred JSON and script responses, dropping the oldest hundred past them', async () => {
        // Given - a session on the site, its script captured
        await using sidecar = await gate();
        const { session } = await sidecar.sessions.create('site');
        await session.page.goto(sidecar.site);

        // When - the page fetches the JSON 499 times, a text and a typeless body, then the JSON once more
        await session.page.evaluate(fetches(1, 499));
        const full = session.captured.map((entry) => entry.url.slice(sidecar.site.length));
        await session.page.evaluate(fetches(500, 500));
        const trimmed = session.captured.map((entry) => entry.url.slice(sidecar.site.length));

        // Then - all 500 kept, then 401: the hundred oldest dropped when the 501st came, the text and the typeless body never captured
        expect({ full: ends(full), trimmed: ends(trimmed) }).toStrictEqual({
            full: { count: 500, first: '/app.js', last: '/data.json?call=499' },
            trimmed: { count: 401, first: '/data.json?call=100', last: '/data.json?call=500' },
        });
    });

    test('closes its context, and a second close is no error', async () => {
        // Given - an open session
        await using sidecar = await gate();
        const { session } = await sidecar.sessions.create('site');

        // When - it is closed twice
        await session.close();
        await session.close();

        // Then - its page went with its context
        expect(session.page.isClosed()).toBe(true);
    });
});
