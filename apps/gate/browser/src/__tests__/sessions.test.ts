import { clock } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { RefusedError } from '../http.ts';
import { gate } from './server.fixtures.ts';

const MINUTE = 60 * 1000;

describe('Sessions', () => {
    test('a provider never synced opens no session and starts no browser', async () => {
        // Given - a sidecar with no cookies for `mail`
        await using sidecar = await gate();

        // When - a session is opened for mail
        const refused = await sidecar.sessions.create('mail').catch((error: unknown) => error);

        // Then - the refusal, and nothing opened
        expect({
            connected: sidecar.sessions.connected,
            refused,
            size: sidecar.sessions.size,
        }).toStrictEqual({
            connected: false,
            refused: new RefusedError(
                500,
                'no cookies for provider "mail" — run the cookie-sync extension',
                'NO_COOKIES',
            ),
            size: 0,
        });
    });

    test('a session’s page is a desktop Chrome, 1280 by 900', async () => {
        // Given - a session
        await using sidecar = await gate();
        const { session } = await sidecar.sessions.create('site');

        // When - its page reads its browser and its viewport
        const seen: unknown = await session.page.evaluate(
            '[navigator.userAgent, innerWidth, innerHeight]',
        );

        // Then - the desktop Chrome a provider's site serves its full page to
        expect(seen).toStrictEqual([
            'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36',
            1280,
            900,
        ]);
    });

    test('reaping ends the expired sessions and keeps the others', async () => {
        // Given - two sessions opened at 08:00, one of them used again at 08:04
        using _ = clock.at('2026-10-09T08:00:00Z');
        await using sidecar = await gate();
        const idle = await sidecar.sessions.create('site');
        const used = await sidecar.sessions.create('site');
        await clock.advance(4 * MINUTE);
        sidecar.sessions.get(used.id);

        // When - the sessions are reaped at 08:06
        await sidecar.sessions.reap(Date.parse('2026-10-09T08:06:00Z'));

        // Then - the idle one ended with its page, the used one still served
        expect({
            idle: idle.session.page.isClosed(),
            size: sidecar.sessions.size,
            used: sidecar.sessions.get(used.id) === used.session,
        }).toStrictEqual({ idle: true, size: 1, used: true });
    });

    test('ending an unknown session is no error, and every session shares one browser', async () => {
        // Given - two sessions
        await using sidecar = await gate();
        const first = await sidecar.sessions.create('site');
        const second = await sidecar.sessions.create('site');

        // When - an unknown session is ended
        await sidecar.sessions.end('nobody');

        // Then - both stand, their pages in one browser
        expect({
            browsers:
                first.session.page.context().browser() === second.session.page.context().browser(),
            connected: sidecar.sessions.connected,
            size: sidecar.sessions.size,
        }).toStrictEqual({ browsers: true, connected: true, size: 2 });
    });

    test('a browser that went away is launched again for the next session', async () => {
        // Given - a session whose browser closed under it
        await using sidecar = await gate();
        const lost = await sidecar.sessions.create('site');
        await lost.session.page.context().browser()?.close();
        const connectedAfterLoss = sidecar.sessions.connected;

        // When - a session is opened
        const next = await sidecar.sessions.create('site');

        // Then - over a new browser, connected
        expect({
            connected: sidecar.sessions.connected,
            connectedAfterLoss,
            fresh: next.session.page.context().browser() !== lost.session.page.context().browser(),
        }).toStrictEqual({ connected: true, connectedAfterLoss: false, fresh: true });
    });

    test('shutting down ends every session and the browser', async () => {
        // Given - two sessions
        await using sidecar = await gate();
        const first = await sidecar.sessions.create('site');
        const second = await sidecar.sessions.create('site');

        // When - the sidecar shuts down
        await sidecar.sessions.shutdown();

        // Then - no session, no browser
        expect({
            closed: [first.session.page.isClosed(), second.session.page.isClosed()],
            connected: sidecar.sessions.connected,
            size: sidecar.sessions.size,
        }).toStrictEqual({ closed: [true, true], connected: false, size: 0 });
    });
});
