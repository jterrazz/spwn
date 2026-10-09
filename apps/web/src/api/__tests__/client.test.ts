import { http, intercept } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import type { World } from '../../domain/model';
import { apiBaseOf, apiGet, apiPost, onConnectionStatusChange, setApiBase } from '../client';
import type { ConnectionStatus } from '../client';
import { NEO, TRINITY } from './client.fixtures';

async function worldsAnswer(payload: Record<string, unknown>[]) {
    setApiBase('http://spwn.test');
    return await intercept(http.get('/api/worlds'), http.json(payload));
}

describe('client', () => {
    test('the worlds route — a declared world is known by its manifest name', async () => {
        // Given - a project that declares a world no container backs
        await using _ = await worldsAnswer([
            { agents: ['neo'], name: 'matrix', status: 'stopped', workspaces: ['.'] },
        ]);

        const [world] = await apiGet<World[]>('/api/worlds');

        // Then - the name stands in for the identity it has no container to carry
        expect(world).toStrictEqual({
            agent: '',
            agents: [{ name: 'neo', role: 'worker', status: 'stopped' }],
            config: 'matrix',
            created_at: '',
            id: 'matrix',
            name: 'matrix',
            status: 'stopped',
            workspaces: [{ name: '.', path: '.' }],
        });
    });

    test('the worlds route — a running world keeps the container identity it was given', async () => {
        // Given - a world with a container behind it
        await using _ = await worldsAnswer([
            {
                agents: [{ name: 'neo', role: 'chief', status: 'running' }],
                config: 'matrix',
                created_at: '2026-01-01T00:00:00Z',
                id: 'world-abc',
                status: 'running',
                workspaces: [{ name: 'default', path: '/src' }],
            },
        ]);

        const [world] = await apiGet<World[]>('/api/worlds');

        // Then - nothing the API stated is replaced
        expect(world).toMatchObject({
            agents: [{ name: 'neo', role: 'chief', status: 'running' }],
            config: 'matrix',
            id: 'world-abc',
            workspaces: [{ name: 'default', path: '/src' }],
        });
    });

    test('the worlds route — the single-valued agent and workspace fields still resolve', async () => {
        // Given - the older shape, one agent and one mount
        await using _ = await worldsAnswer([
            {
                agent: 'neo',
                config: 'matrix',
                created_at: '2026-01-01T00:00:00Z',
                id: 'world-abc',
                status: 'idle',
                workspace: '/src',
            },
        ]);

        const [world] = await apiGet<World[]>('/api/worlds');

        // Then - both are lifted into the plural form the UI reads
        expect(world).toMatchObject({
            agent: 'neo',
            agents: [{ name: 'neo', role: 'worker', status: 'idle' }],
            workspaces: [{ name: 'default', path: '/src' }],
        });
    });

    test("the worlds route — missing fields take the world's status and the legacy mount", async () => {
        // Given - agent records without a status, an empty workspace list beside a legacy mount, a world with neither, and an empty mount
        await using _ = await worldsAnswer([
            {
                agents: [{ name: 'neo' }],
                id: 'a',
                status: 'running',
                workspace: '/src',
                workspaces: [],
            },
            { id: 'b', status: 'idle' },
            { id: 'c', workspace: '' },
        ]);

        const worlds = await apiGet<World[]>('/api/worlds');

        // Then - each field filled from what the world does state
        expect(
            worlds.map((world) => [world.agents, world.workspaces, world.status, world.config]),
        ).toStrictEqual([
            [
                [{ name: 'neo', role: 'worker', status: 'running' }],
                [{ name: 'default', path: '/src' }],
                'running',
                '',
            ],
            [[], undefined, 'idle', ''],
            [[], undefined, 'idle', ''],
        ]);
    });

    test('any other route, and a worlds answer that is no list, come back as the API wrote them', async () => {
        // Given - the base the app was given, a teams list, and a worlds route answering no list
        setApiBase('http://spwn.test');
        await using _ = await intercept([
            {
                request: http.get('http://spwn.test/api/teams'),
                response: http.json([{ slug: 'ops' }]),
            },
            {
                request: http.get('http://spwn.test/api/worlds'),
                response: http.json({ worlds: 'not ready' }),
            },
        ]);

        // Then - both untouched
        expect([await apiGet('/api/teams'), await apiGet('/api/worlds')]).toStrictEqual([
            [{ slug: 'ops' }],
            { worlds: 'not ready' },
        ]);
    });

    test('a post sends its body as JSON, and a refusal rejects with the reason the API gave', async () => {
        // Given - a route that echoes what it received, one that refuses with a reason, one without
        setApiBase('http://spwn.test');
        await using _ = await intercept([
            {
                request: http.post('http://spwn.test/api/echo'),
                response: (request) =>
                    http.json({
                        body: request.body,
                        method: request.method,
                        type: request.headers['content-type'],
                    }),
            },
            {
                request: http.post('http://spwn.test/api/refuse'),
                response: http.error(500, { error: 'boom' }),
            },
            {
                request: http.post('http://spwn.test/api/mute'),
                response: http.error(500, { message: 'no error field' }),
            },
        ]);

        // Then - the echo of the post, the refusal's reason, and the status when it gives none
        await expect(apiPost('/api/echo', { name: 'neo' })).resolves.toStrictEqual({
            body: { name: 'neo' },
            method: 'POST',
            type: 'application/json',
        });
        await expect(apiPost('/api/refuse')).rejects.toMatchObject({
            message: 'boom',
            name: 'ApiError',
        });
        await expect(apiPost('/api/mute')).rejects.toThrow('API error 500');
    });

    test('an aborted request fails as an unreachable API', async () => {
        // Given - a reachable route and a signal aborted before the call
        setApiBase('http://spwn.test');
        await using _ = await intercept(http.post('http://spwn.test/api/echo'), http.json({}));
        const controller = new AbortController();
        controller.abort();

        // Then - the abort wins over the answer
        await expect(apiPost('/api/echo', {}, { signal: controller.signal })).rejects.toMatchObject(
            {
                cause: { name: 'AbortError' },
                message: 'Failed to connect to API',
            },
        );
    });

    test('the connection status is announced once per change', async () => {
        // Given - a connected app, then two refusals and two answers
        setApiBase('http://spwn.test');
        await using _ = await intercept([
            { request: http.get('http://spwn.test/api/up'), response: http.json({}) },
            { request: http.get('http://spwn.test/api/down'), response: http.error(503) },
            { request: http.get('http://spwn.test/api/gone'), response: http.unreachable() },
        ]);
        await apiGet('/api/up');
        const heard: ConnectionStatus[] = [];
        const stop = onConnectionStatusChange((status) => {
            heard.push(status);
        });

        // When - the API refuses twice, answers twice, is unreachable, then refuses once the listener left
        await expect(apiGet('/api/down')).rejects.toThrow('HTTP 503');
        await expect(apiGet('/api/down')).rejects.toThrow('HTTP 503');
        await apiGet('/api/up');
        await apiGet('/api/up');
        await expect(apiGet('/api/gone')).rejects.toThrow('Failed to connect to API');
        stop();
        await apiGet('/api/up');

        // Then - one announcement per change while it listened
        expect(heard).toStrictEqual(['disconnected', 'connected', 'disconnected']);
    });

    test('an agent route reads the profile with its defaults, other routes keep their fields', async () => {
        // Given - a whole profile, one whose journal the API wrote as null, a team route, and an agent route answering no name
        setApiBase('http://spwn.test');
        await using _ = await intercept([
            {
                request: http.get('http://spwn.test/api/agents/neo'),
                response: http.json(NEO),
            },
            {
                request: http.get('http://spwn.test/api/agents/trinity'),
                response: http.json({ journal: null, name: 'trinity' }),
            },
            {
                request: http.get('http://spwn.test/api/teams/ops'),
                response: http.json({ name: 'Ops' }),
            },
            {
                request: http.get('http://spwn.test/api/agents/ghost'),
                response: http.json({ error: 'none' }),
            },
        ]);

        // Then - the whole profile, then both other answers untouched
        expect([
            await apiGet('/api/agents/neo'),
            await apiGet('/api/agents/trinity'),
            await apiGet('/api/teams/ops'),
            await apiGet('/api/agents/ghost'),
        ]).toStrictEqual([NEO, TRINITY, { name: 'Ops' }, { error: 'none' }]);
    });

    const NONE = { build: undefined, desktop: null, given: null, host: undefined };

    test.each([
        {
            base: 'http://given.test',
            sources: { ...NONE, desktop: 'http://localhost:4100', given: 'http://given.test' },
        },
        {
            base: 'http://localhost:4100',
            sources: { ...NONE, build: 'http://build.test', desktop: 'http://localhost:4100' },
        },
        {
            base: 'http://build.test',
            sources: { ...NONE, build: 'http://build.test', host: 'box.local' },
        },
        { base: 'http://box.local:3001', sources: { ...NONE, build: '', host: 'box.local' } },
        { base: 'http://box.local:3001', sources: { ...NONE, host: 'box.local' } },
        { base: 'http://localhost:3001', sources: { ...NONE, given: '', host: '' } },
        { base: 'http://localhost:3001', sources: NONE },
    ])(
        'the API base is $base when the first source that names one says so',
        ({ base, sources }) => {
            // Given - what the app was given, the desktop shell, the build and the page each say

            // Then - the first that names a base, localhost when none does
            expect(apiBaseOf(sources)).toBe(base);
        },
    );
});
