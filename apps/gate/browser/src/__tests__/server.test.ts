import { describe, expect, test } from 'vitest';

import { gate, idOf } from './server.fixtures.ts';

const unrouted = { body: { error: 'no route' }, status: 404 };

describe('sidecar', () => {
    test('a path is a route only whole, never inside a longer one', async () => {
        // Given - a sidecar
        await using sidecar = await gate();

        // When - each route is asked under a prefix, then with a suffix
        const routes = [
            ['POST', '/sessions'],
            ['DELETE', '/sessions/a'],
            ...[
                'goto',
                'click',
                'type',
                'scroll',
                'wait-selector',
                'wait-response',
                'eval',
                'captured-responses',
            ].map((action) => ['POST', `/sessions/a/${action}`]),
            ['GET', '/healthz'],
        ];
        const replies = await Promise.all(
            routes.flatMap(([method = '', path = '']) => [
                sidecar.call(method, `/v1${path}`),
                sidecar.call(method, `${path}/more`),
            ]),
        );

        // Then - none of them routed
        expect(replies).toStrictEqual(routes.flatMap(() => [unrouted, unrouted]));
    });

    test('a request no route answers is refused, and a query string is not part of the path', async () => {
        // Given - a sidecar
        await using sidecar = await gate();

        // When - an unknown path, a known path by the wrong method, and the health with a query are asked
        const replies = [
            await sidecar.call('GET', '/nowhere'),
            await sidecar.call('GET', '/sessions'),
            await sidecar.call('GET', '/healthz?probe=1'),
        ];

        // Then - two refusals, then the health
        expect(replies).toStrictEqual([
            { body: { error: 'no route' }, status: 404 },
            { body: { error: 'no route' }, status: 404 },
            { body: { browser_connected: false, ok: true, sessions: 0 }, status: 200 },
        ]);
    });

    test('a body that is no JSON is refused, and JSON that is no object reads as empty', async () => {
        // Given - a sidecar
        await using sidecar = await gate();

        // When - a session is asked with a broken body, an array, a string, an empty one, and an object naming no provider
        const replies = [
            await sidecar.call('POST', '/sessions', undefined, '{"provider":'),
            await sidecar.call('POST', '/sessions', undefined, '["site"]'),
            await sidecar.call('POST', '/sessions', undefined, '"site"'),
            await sidecar.call('POST', '/sessions'),
            await sidecar.call('POST', '/sessions', {}),
        ];

        // Then - the parse error named, then four bodies with no provider
        const unnamed = { body: { code: null, error: 'provider is required' }, status: 400 };
        expect(replies).toStrictEqual([
            {
                body: {
                    code: null,
                    error: `bad json: Unexpected end of JSON input`,
                },
                status: 400,
            },
            unnamed,
            unnamed,
            unnamed,
            unnamed,
        ]);
    });

    test('a DELETE reads no body, whatever it carries', async () => {
        // Given - a session
        await using sidecar = await gate();
        const { body } = await sidecar.call('POST', '/sessions', { provider: 'site' });

        // When - it is ended with a body that is no JSON
        const reply = await sidecar.call(
            'DELETE',
            `/sessions/${idOf(body)}`,
            undefined,
            '{"provider":',
        );

        // Then - ended, the body never parsed
        expect(reply).toStrictEqual({ body: '', status: 204 });
    });

    test('a failure that is no refusal is answered 500, with its message and no code', async () => {
        // Given - a session
        await using sidecar = await gate();
        const { body } = await sidecar.call('POST', '/sessions', { provider: 'site' });
        const id = idOf(body);

        // When - its page evaluates a script that throws
        const reply = await sidecar.call('POST', `/sessions/${id}/eval`, {
            script: 'Promise.reject("boom")',
        });

        // Then - the browser's error, answered as the sidecar's own failure
        expect(reply).toStrictEqual({
            body: { code: null, error: 'page.evaluate: boom' },
            status: 500,
        });
    });
});
