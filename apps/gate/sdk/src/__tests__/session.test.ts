import { once } from 'node:events';
import { createServer } from 'node:http';
import type { Server } from 'node:http';
import { describe, expect, test, vi } from 'vitest';

import { openSession } from '../session.ts';

type Received = { body: unknown; method: string | undefined; url: string | undefined };

async function sidecar(
    status: number,
    reply: (url: string | undefined) => unknown,
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
                url: request.url,
            });
            response
                .writeHead(status, { 'content-type': 'application/json' })
                .end(JSON.stringify(reply(request.url)));
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

describe('Session', () => {
    test('a session drives the sidecar route by route, and closes once', async () => {
        // Given - a sidecar that opens session-1 and acknowledges every action
        const { origin, received, server } = await sidecar(200, (url) =>
            url === '/sessions' ? { id: 'session-1' } : { ok: true },
        );
        vi.stubEnv('GATE_BROWSER_URL', origin);

        // When - a tool runs every action of a session, then ends it twice
        const session = await openSession('x');
        await session.goto('https://x.com/home', { wait_until: 'networkidle' });
        await session.waitResponse('/Bookmarks', { method: 'POST' });
        await session.click('[data-testid="tweetButton"]');
        await session.type('textarea', 'hello');
        await session.scroll({ count: 2, wait_ms: 0 });
        await session.waitSelector('[data-ready="true"]');
        await session.eval('document.title');
        await session.capturedResponses({ url_pattern: 'GraphQL' });
        await session.end();
        await session.end();
        server.close();

        // Then - the requests the sidecar received, in order
        expect(received).toStrictEqual([
            { body: { provider: 'x' }, method: 'POST', url: '/sessions' },
            {
                body: { url: 'https://x.com/home', wait_until: 'networkidle' },
                method: 'POST',
                url: '/sessions/session-1/goto',
            },
            {
                body: { method: 'POST', url_pattern: '/Bookmarks' },
                method: 'POST',
                url: '/sessions/session-1/wait-response',
            },
            {
                body: { selector: '[data-testid="tweetButton"]' },
                method: 'POST',
                url: '/sessions/session-1/click',
            },
            {
                body: { selector: 'textarea', text: 'hello' },
                method: 'POST',
                url: '/sessions/session-1/type',
            },
            { body: { count: 2, wait_ms: 0 }, method: 'POST', url: '/sessions/session-1/scroll' },
            {
                body: { selector: '[data-ready="true"]' },
                method: 'POST',
                url: '/sessions/session-1/wait-selector',
            },
            { body: { script: 'document.title' }, method: 'POST', url: '/sessions/session-1/eval' },
            {
                body: { url_pattern: 'GraphQL' },
                method: 'POST',
                url: '/sessions/session-1/captured-responses',
            },
            { body: null, method: 'DELETE', url: '/sessions/session-1' },
        ]);
    });

    test('a sidecar refusal fails the call with its method, URL, status and reason', async () => {
        // Given - a sidecar answering 503 with its reason
        const { origin, server } = await sidecar(503, () => ({ error: 'sidecar unavailable' }));
        vi.stubEnv('GATE_BROWSER_URL', origin);

        // Then - the open fails with every part of the refusal
        await expect(openSession('x')).rejects.toThrow(
            `POST ${origin}/sessions → 503: sidecar unavailable`,
        );
        server.close();
    });
});
