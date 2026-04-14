import { describe, expect, test } from 'vitest';

/**
 * GET /api/activity contract shape.
 *
 * The legacy file conditionally ran HTTP tests against a live Go API
 * server (`GO_API_URL`). Those paths were skipped on CI and in local
 * `pnpm test` runs, so all that actually executed were the two
 * server-independent shape assertions below. The server-coupled
 * checks belong in a Go integration test (`packages/server/api/...`)
 * and are intentionally not ported here.
 */

interface ActivityEvent {
    actor: string;
    agent_id?: string;
    cost_usd?: number;
    duration_ms?: number;
    id: string;
    phrase: string;
    target?: string;
    timestamp: string;
    type: string;
    verb: string;
    world_id?: string;
}

describe('GET /api/activity contract', () => {
    test('response shape has events array of well-formed entries', () => {
        const validResponse: { events: ActivityEvent[] } = {
            events: [
                {
                    actor: 'user',
                    agent_id: 'neo',
                    id: 'abc123def456',
                    phrase: 'You created neo',
                    target: 'neo',
                    timestamp: '2026-04-04T12:00:00.000Z',
                    type: 'agent.created',
                    verb: 'created',
                },
            ],
        };

        expect(validResponse).toHaveProperty('events');
        expect(Array.isArray(validResponse.events)).toBe(true);

        const event = validResponse.events[0];
        expect(event).toHaveProperty('id');
        expect(event).toHaveProperty('timestamp');
        expect(event).toHaveProperty('type');
        expect(event).toHaveProperty('actor');
        expect(event).toHaveProperty('verb');
        expect(event).toHaveProperty('phrase');
    });

    test('event type uses dotted namespace', () => {
        const validTypes = [
            'world.spawned',
            'world.destroyed',
            'world.snapshot',
            'world.session_ended',
            'agent.created',
            'agent.deleted',
            'agent.joined',
            'agent.left',
            'agent.dreamed',
            'agent.slept',
            'agent.forked',
            'agent.talked',
            'architect.started',
            'architect.stopped',
            'architect.talked',
        ];

        for (const type of validTypes) {
            expect(type).toMatch(/^[a-z]+\.[a-z_]+$/);
        }
    });
});
