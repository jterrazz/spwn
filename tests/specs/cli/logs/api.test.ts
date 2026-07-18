import { expect, test } from 'vitest';

/**
 * GET /api/activity contract shape. The server-coupled paths in the legacy
 * suite were skipped on CI and belong in a Go integration test; only the two
 * server-independent shape assertions actually ran, and they are ported here
 * verbatim. No CLI is exercised, so no result binds with `await using`.
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

test('response shape has an events array of well-formed entries', () => {
    // Given - a representative activity response payload
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

    // Then - it carries an events array whose entries expose the required fields
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

test('event type uses a dotted namespace', () => {
    // Given - the full set of emitted event types
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

    // Then - each is a lowercase dotted namespace
    for (const type of validTypes) {
        expect(type).toMatch(/^[a-z]+\.[a-z_]+$/);
    }
});
