import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { afterAll, afterEach, beforeAll, describe, expect, test } from 'vitest';

import type { World } from '@/domain/model';

import { apiGet, setApiBase } from './client';

const server = setupServer();

beforeAll(() => {
    setApiBase('http://spwn.test');
    server.listen({ onUnhandledRequest: 'error' });
});

afterEach(() => {
    server.resetHandlers();
});

afterAll(() => {
    server.close();
});

function worldsAnswer(payload: Record<string, unknown>[]) {
    server.use(http.get('http://spwn.test/api/worlds', () => HttpResponse.json(payload)));
}

describe('the worlds route', () => {
    test('a declared world is known by its manifest name', async () => {
        // Given - a project that declares a world no container backs
        worldsAnswer([{ agents: ['neo'], name: 'matrix', status: 'stopped', workspaces: ['.'] }]);

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

    test('a running world keeps the container identity it was given', async () => {
        // Given - a world with a container behind it
        worldsAnswer([
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

    test('the single-valued agent and workspace fields still resolve', async () => {
        // Given - the older shape, one agent and one mount
        worldsAnswer([
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
});
