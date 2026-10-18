import { http, intercept } from '@jterrazz/test/stubs';
import { originalOf } from '@jterrazz/typescript/rules';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { mkdirSync, mkdtempSync, rmSync, symlinkSync } from 'node:fs';
import { createServer } from 'node:http';
import type { Server } from 'node:http';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
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

// outside: a mutation run copies the member into a sandbox, where the repository's catalog is not
const X_TOOL = resolve(
    originalOf(import.meta.dirname),
    '../../../../../catalog/x/tools/x/index.js',
);

// outside: the gate hands a catalog tool this SDK through NODE_PATH, as @spwn/gate-tool
function gateNodePath() {
    const folder = mkdtempSync(join(tmpdir(), 'gate-node-path-'));
    mkdirSync(join(folder, '@spwn'));
    symlinkSync(resolve(import.meta.dirname, '../..'), join(folder, '@spwn', 'gate-tool'));
    return {
        folder,
        [Symbol.dispose]() {
            rmSync(folder, { force: true, recursive: true });
        },
    };
}

async function stdoutOf(args: string[], env: NodeJS.ProcessEnv): Promise<string> {
    const child = spawn('node', args, { env });
    const chunks: Buffer[] = [];
    child.stdout.on('data', (chunk: Buffer) => {
        chunks.push(chunk);
    });
    await once(child, 'close');
    return Buffer.concat(chunks).toString('utf8');
}

const CREATED = {
    body: { data: { create_tweet: { tweet_results: { result: { rest_id: '42' } } } } },
    status: 200,
    url: 'https://x.com/i/api/graphql/q/CreateTweet',
};

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

    test('a session opens on the sidecar the gate publishes at 127.0.0.1:9001 unless told another', async () => {
        // Given - no sidecar address in the environment, and the published address answering an open
        vi.stubEnv('GATE_BROWSER_URL', undefined);
        await using _ = await intercept([
            {
                request: http.post('http://127.0.0.1:9001/sessions'),
                response: http.json({ id: 'session-1' }),
            },
            {
                request: http.delete('http://127.0.0.1:9001/sessions/session-1'),
                response: http.json({}),
            },
        ]);

        // Then - the open and the close both reached it, and nowhere else
        const session = await openSession('x');
        await expect(session.end()).resolves.toBeUndefined();
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

    test('the catalog’s x tool posts a tweet through this session, each call one the sidecar routes', async () => {
        // Given - the x tool handed this SDK as the gate hands it, and a sidecar that sees the tweet created
        using nodePath = gateNodePath();
        const { origin, received, server } = await sidecar(200, (url) => {
            if (url === '/sessions') {
                return JSON.stringify({ id: 'x-1' });
            }
            return url === '/sessions/x-1/wait-response' ? JSON.stringify(CREATED) : '';
        });

        // When - the tool's post-tweet runs as the gate runs it
        const { NODE_OPTIONS: _, ...env } = process.env;
        const stdout = await stdoutOf([X_TOOL, 'post-tweet', '--text', 'hello'], {
            ...env,
            GATE_BROWSER_URL: origin,
            NODE_PATH: nodePath.folder,
        });
        server.close();

        // Then - the session opened, browsed, typed, awaited the creation, clicked, ended, and the tweet it reports
        const composer = '[data-testid="tweetTextarea_0"]';
        const button = '[data-testid="tweetButton"]';
        const path = '/sessions/x-1';
        const [opened, ...browsed] = received;
        const raced = browsed
            .slice(4, 6)
            .toSorted((left, right) => String(left.url).localeCompare(String(right.url)));
        // invariant: the tool awaits the creation and clicks at once, so those two reach the sidecar in either order
        const ordered = [opened, ...browsed.slice(0, 4), ...raced, ...browsed.slice(6)];
        expect({ sent: ordered, stdout: JSON.parse(stdout) as unknown }).toStrictEqual({
            sent: [
                sent('/sessions', { provider: 'x' }),
                sent(`${path}/goto`, {
                    timeout_ms: 25_000,
                    url: 'https://x.com/compose/post',
                    wait_until: 'domcontentloaded',
                }),
                sent(`${path}/wait-selector`, { selector: composer, timeout_ms: 12_000 }),
                sent(`${path}/type`, { selector: composer, text: 'hello' }),
                sent(`${path}/wait-selector`, { selector: button, timeout_ms: 5000 }),
                sent(`${path}/click`, { selector: button }),
                sent(`${path}/wait-response`, {
                    method: 'POST',
                    timeout_ms: 20_000,
                    url_pattern: '/CreateTweet',
                }),
                { body: null, method: 'DELETE', type: 'application/json', url: path },
            ],
            stdout: { external_id: '42', url: 'https://x.com/i/web/status/42', user: null },
        });
    });

    test('a session needs a provider to load its cookies from', async () => {
        // Given - no provider named

        // Then - the open is refused before any request
        await expect(openSession('')).rejects.toThrow('openSession: provider name required');
    });
});
