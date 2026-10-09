import { once } from 'node:events';
import { createServer } from 'node:http';
import type { Server } from 'node:http';
import { describe, expect, test, vi } from 'vitest';

import { openSession } from '../session.ts';

type Received = {
    body: unknown;
    method: string | undefined;
    type: string | undefined;
    url: string | undefined;
};

async function sidecar(
    status: number,
    reply: (url: string | undefined) => string,
): Promise<{ origin: string; received: Received[]; server: Server }> {
    const received: Received[] = [];
    const server = createServer((request, response) => {
        const chunks: Buffer[] = [];
        request.on('data', (chunk: Buffer) => {
            chunks.push(chunk);
        });
        request.on('end', () => {
            const raw = Buffer.concat(chunks).toString('utf8');
            received.push({
                body: raw === '' ? null : (JSON.parse(raw) as unknown),
                method: request.method,
                type: request.headers['content-type'],
                url: request.url,
            });
            response.writeHead(status).end(reply(request.url));
        });
    });
    server.listen(0, '127.0.0.1');
    await once(server, 'listening');
    const address = server.address();
    return {
        origin: `http://127.0.0.1:${typeof address === 'object' && address !== null ? address.port : 0}`,
        received,
        server,
    };
}

const JSON_TYPE = 'application/json';

async function driven() {
    const { origin, received, server } = await sidecar(200, (url) => {
        if (url === '/sessions') {
            return JSON.stringify({ id: 'session-1' });
        }
        return url === '/sessions/session-1/eval' ? JSON.stringify('Home / X') : '';
    });
    vi.stubEnv('GATE_BROWSER_URL', origin);
    const session = await openSession('x');
    const results = [
        await session.goto('https://x.com/home', { wait_until: 'networkidle' }),
        await session.waitResponse('/Bookmarks', { method: 'POST' }),
        await session.click('[data-testid="tweetButton"]'),
        await session.type('textarea', 'hello'),
        await session.scroll({ count: 2, wait_ms: 0 }),
        await session.waitSelector('[data-ready="true"]'),
        await session.eval('document.title'),
        await session.capturedResponses({ url_pattern: 'GraphQL' }),
    ];
    await session.end();
    await session.end();
    server.close();
    return { received, results };
}

function sent(url: string, body: unknown, method = 'POST'): Received {
    return { body, method, type: JSON_TYPE, url };
}

describe('Session', () => {
    test('each action returns what the sidecar answered, nothing for an empty answer', async () => {
        // Given - a sidecar that evaluates to the page title and answers every other action empty
        const { results } = await driven();

        // Then - the title for eval, null for the rest
        expect(results).toStrictEqual([null, null, null, null, null, null, 'Home / X', null]);
    });

    test('a session drives the sidecar route by route, and closes once', async () => {
        // Given - a tool that runs every action of a session, then ends it twice
        const { received } = await driven();

        // Then - the requests the sidecar received, in order, each a JSON body
        expect(received).toStrictEqual([
            sent('/sessions', { provider: 'x' }),
            sent('/sessions/session-1/goto', {
                url: 'https://x.com/home',
                wait_until: 'networkidle',
            }),
            sent('/sessions/session-1/wait-response', {
                method: 'POST',
                url_pattern: '/Bookmarks',
            }),
            sent('/sessions/session-1/click', { selector: '[data-testid="tweetButton"]' }),
            sent('/sessions/session-1/type', { selector: 'textarea', text: 'hello' }),
            sent('/sessions/session-1/scroll', { count: 2, wait_ms: 0 }),
            sent('/sessions/session-1/wait-selector', { selector: '[data-ready="true"]' }),
            sent('/sessions/session-1/eval', { script: 'document.title' }),
            sent('/sessions/session-1/captured-responses', { url_pattern: 'GraphQL' }),
            sent('/sessions/session-1', null, 'DELETE'),
        ]);
    });

    test.each([
        {
            body: JSON.stringify({ error: 'bad provider' }),
            reason: 'bad provider',
            status: 400,
        },
        { body: JSON.stringify({ error: 503 }), reason: '{"error":503}', status: 503 },
        { body: 'upstream down', reason: 'upstream down', status: 502 },
    ])(
        'a sidecar refusal $status fails the call with its method, URL, status and reason',
        async ({ body, reason, status }) => {
            // Given - a sidecar refusing with a JSON reason, a reason that is not text, or plain text
            const { origin, server } = await sidecar(status, () => body);
            vi.stubEnv('GATE_BROWSER_URL', origin);

            // Then - the open fails with the reason when it is text, the whole body otherwise
            await expect(openSession('x')).rejects.toThrow(
                `POST ${origin}/sessions → ${status}: ${reason}`,
            );
            server.close();
        },
    );

    test('a session needs a provider to load its cookies from', async () => {
        // Given - no provider named

        // Then - the open is refused before any request
        await expect(openSession('')).rejects.toThrow('openSession: provider name required');
    });
});
