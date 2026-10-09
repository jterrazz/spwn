import { clock } from '@jterrazz/test';
import { stripVTControlCharacters } from 'node:util';
import { describe, expect, test } from 'vitest';

import { gate, idOf } from './server.fixtures.ts';

const ok = { body: { ok: true }, status: 200 };

function refusalOf(reply: { body: unknown; status: number }) {
    const error: unknown =
        typeof reply.body === 'object' && reply.body !== null
            ? Reflect.get(reply.body, 'error')
            : '';
    const lines = stripVTControlCharacters(String(error)).split('\n');
    return { error: lines.slice(0, 3).map((line) => line.trim()), status: reply.status };
}

async function opened(sidecar: Awaited<ReturnType<typeof gate>>): Promise<string> {
    const { body } = await sidecar.call('POST', '/sessions', { provider: 'site' });
    return idOf(body);
}

describe('ROUTES', () => {
    test('a session opens on its provider’s cookies, drives its page, then ends', async () => {
        // Given - a sidecar whose provider `site` holds a cookie for localhost, at a known instant
        using _ = clock.at('2026-10-09T08:00:00Z');
        await using sidecar = await gate();

        // When - a tool opens a session, browses the site, types, clicks, waits and reads, then ends it
        const created = await sidecar.call('POST', '/sessions', { provider: 'site' });
        const id = idOf(created.body);
        const path = `/sessions/${id}`;
        const replies = [
            await sidecar.call('POST', `${path}/goto`, { url: sidecar.site }),
            await sidecar.call('POST', `${path}/eval`, { script: 'document.cookie' }),
            await sidecar.call('POST', `${path}/type`, { selector: '#name', text: 'neo' }),
            await sidecar.call('POST', `${path}/eval`, {
                script: "document.querySelector('#name').value",
            }),
            await sidecar.call('POST', `${path}/click`, { selector: '#load' }),
            await sidecar.call('POST', `${path}/wait-selector`, {
                selector: '#out[data-ready="true"]',
            }),
            await sidecar.call('POST', `${path}/eval`, {
                script: "document.querySelector('#out').textContent",
            }),
            await sidecar.call('DELETE', path),
            await sidecar.call('POST', `${path}/eval`, { script: '1' }),
        ];

        // Then - the session's id and deadline, each action's answer, and the session unknown once ended
        expect(created).toStrictEqual({
            body: {
                expires_at: '2026-10-09T08:30:00.000Z',
                id,
                provider: 'site',
            },
            status: 201,
        });
        expect(replies).toStrictEqual([
            { body: { ok: true, status: 200, url: `${sidecar.site}/` }, status: 200 },
            { body: { result: 'token=abc' }, status: 200 },
            ok,
            { body: { result: 'neo' }, status: 200 },
            ok,
            ok,
            { body: { result: '{"items":[1,2]}' }, status: 200 },
            { body: '', status: 204 },
            { body: { code: null, error: `unknown session "${id}"` }, status: 404 },
        ]);
    });

    test('a type with no text is refused, and an empty text types nothing', async () => {
        // Given - a session on the site
        await using sidecar = await gate();
        const path = `/sessions/${await opened(sidecar)}`;
        await sidecar.call('POST', `${path}/goto`, { url: sidecar.site });

        // When - a tool types without a text, a text that is no string, then an empty one
        const replies = [
            await sidecar.call('POST', `${path}/type`, { selector: '#name' }),
            await sidecar.call('POST', `${path}/type`, { selector: '#name', text: 7 }),
            await sidecar.call('POST', `${path}/type`, { selector: '#name', text: '' }),
            await sidecar.call('POST', `${path}/eval`, {
                script: "document.querySelector('#name').value",
            }),
        ];

        // Then - two refusals, then an accepted empty text that left the field empty
        const refused = { body: { code: null, error: 'text is required' }, status: 400 };
        expect(replies).toStrictEqual([
            refused,
            refused,
            ok,
            { body: { result: '' }, status: 200 },
        ]);
    });

    test('an action missing what it acts on is refused, each by name', async () => {
        // Given - a session on the site
        await using sidecar = await gate();
        const path = `/sessions/${await opened(sidecar)}`;

        // When - each action is sent an empty body
        const actions = ['goto', 'click', 'type', 'wait-selector', 'wait-response', 'eval'];
        const replies = await Promise.all(
            actions.map(async (action) => await sidecar.call('POST', `${path}/${action}`, {})),
        );

        // Then - the field each one needs, named
        expect(replies).toStrictEqual(
            ['url', 'selector', 'selector', 'selector', 'url_pattern', 'script'].map((field) => ({
                body: { code: null, error: `${field} is required` },
                status: 400,
            })),
        );
    });

    test('goto waits for the load state asked and answers the page’s own status', async () => {
        // Given - a session
        await using sidecar = await gate();
        const path = `/sessions/${await opened(sidecar)}`;

        // When - it goes to a missing page waiting for load, to the site with an unknown state, to an anchor of it, then waits for the network to idle in 1 ms
        const replies = [
            await sidecar.call('POST', `${path}/goto`, {
                timeout_ms: 5000,
                url: `${sidecar.site}/gone`,
                wait_until: 'load',
            }),
            await sidecar.call('POST', `${path}/goto`, { url: sidecar.site, wait_until: 'soon' }),
            await sidecar.call('POST', `${path}/goto`, { url: `${sidecar.site}/#end` }),
            await sidecar.call('POST', `${path}/goto`, {
                timeout_ms: 1,
                url: sidecar.site,
                wait_until: 'networkidle',
            }),
        ];

        // Then - the 404 answered as the page's, the unknown state read as the default, an anchor answering no response, the budget exceeded
        expect({
            answered: replies.slice(0, 3),
            refused: refusalOf(replies[3] ?? ok),
        }).toStrictEqual({
            answered: [
                { body: { ok: true, status: 404, url: `${sidecar.site}/gone` }, status: 200 },
                { body: { ok: true, status: 200, url: `${sidecar.site}/` }, status: 200 },
                { body: { ok: true, status: null, url: `${sidecar.site}/#end` }, status: 200 },
            ],
            refused: {
                error: [
                    'page.goto: Timeout 1ms exceeded.',
                    'Call log:',
                    `- navigating to "${sidecar.site}/", waiting until "networkidle"`,
                ],
                status: 500,
            },
        });
    });

    test('a scroll wheels the page as many times as asked', async () => {
        // Given - a session on the tall site
        await using sidecar = await gate();
        const path = `/sessions/${await opened(sidecar)}`;
        await sidecar.call('POST', `${path}/goto`, { url: sidecar.site });

        // When - it scrolls twice by 300, then with a count and a delta of zero
        const replies = [
            await sidecar.call('POST', `${path}/scroll`, { count: 2, delta_y: 300, wait_ms: 50 }),
            await sidecar.call('POST', `${path}/eval`, { script: 'window.scrollY' }),
            await sidecar.call('POST', `${path}/scroll`, { count: 0, delta_y: 0, wait_ms: 50 }),
            await sidecar.call('POST', `${path}/eval`, { script: 'window.scrollY' }),
        ];

        // Then - 600 pixels down, then 4000 more once, zero read as the defaults
        expect(replies).toStrictEqual([
            ok,
            { body: { result: 600 }, status: 200 },
            ok,
            { body: { result: 4600 }, status: 200 },
        ]);
    });

    test('a selector is awaited in the state asked, visible by default', async () => {
        // Given - a session on the site, its output empty
        await using sidecar = await gate();
        const path = `/sessions/${await opened(sidecar)}`;
        await sidecar.call('POST', `${path}/goto`, { url: sidecar.site });

        // When - a tool awaits the output attached, a missing node detached, then the empty output visible
        const replies = [
            await sidecar.call('POST', `${path}/wait-selector`, {
                selector: '#out',
                state: 'attached',
            }),
            await sidecar.call('POST', `${path}/wait-selector`, {
                selector: '#none',
                state: 'detached',
            }),
            await sidecar.call('POST', `${path}/wait-selector`, {
                selector: '#out',
                state: 'shown',
                timeout_ms: 300,
            }),
        ];

        // Then - both found, and an empty block never visible: an unknown state read as visible
        expect({ found: replies.slice(0, 2), refused: refusalOf(replies[2] ?? ok) }).toStrictEqual({
            found: [ok, ok],
            refused: {
                error: [
                    'page.waitForSelector: Timeout 300ms exceeded.',
                    'Call log:',
                    "- waiting for locator('#out') to be visible",
                ],
                status: 500,
            },
        });
    });

    test('a click and a type wait for their element no longer than asked', async () => {
        // Given - a session on the site
        await using sidecar = await gate();
        const path = `/sessions/${await opened(sidecar)}`;
        await sidecar.call('POST', `${path}/goto`, { url: sidecar.site });

        // When - a tool clicks, then types into, an element the page does not hold, within 300 ms
        const replies = [
            await sidecar.call('POST', `${path}/click`, { selector: '#none', timeout_ms: 300 }),
            await sidecar.call('POST', `${path}/type`, {
                selector: '#none',
                text: 'x',
                timeout_ms: 300,
            }),
        ];

        // Then - both refused when their budget ran out
        const refused = {
            error: [
                'page.click: Timeout 300ms exceeded.',
                'Call log:',
                "- waiting for locator('#none')",
            ],
            status: 500,
        };
        expect(replies.map(refusalOf)).toStrictEqual([refused, refused]);
    });

    test('health counts the open sessions and the warm browser', async () => {
        // Given - a sidecar with one session open
        await using sidecar = await gate();
        await opened(sidecar);

        // When - the health is read
        const health = await sidecar.call('GET', '/healthz');

        // Then - one session, over a connected browser
        expect(health).toStrictEqual({
            body: { browser_connected: true, ok: true, sessions: 1 },
            status: 200,
        });
    });
});
