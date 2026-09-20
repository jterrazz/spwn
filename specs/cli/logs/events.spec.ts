import { describe, expect, test } from 'vitest';

import { required } from '../../_support/required.js';
import { cli } from '../cli.specification.js';

/**
 * Activity event emissions — the specs that read `activity.jsonl` as DATA
 * rather than as text: one entry of a given type, a metadata field, ids
 * compared to each other, timestamps compared to each other. A document's
 * `files: { equals }` is byte-exact text and has no vocabulary for any of
 * them; the whole-file shape it CAN carry is stated in
 * `activity-log.spec.yaml` beside this file, with `{{hex}}` ids and
 * `{{iso8601}}` timestamps.
 *
 * Each spec uses an isolated `$WORKDIR/spwn-home` so its log starts empty.
 * Every result binds with `await using` (rule B5).
 */

const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

interface ActivityEvent {
    actor: string;
    agent_id?: string;
    id: string;
    metadata?: Record<string, unknown>;
    phrase: string;
    target?: string;
    timestamp: string;
    type: string;
    verb: string;
    world_id?: string;
}

function parseActivity(content: string): ActivityEvent[] {
    return content
        .split('\n')
        .filter((line) => line.trim())
        .map((line) => JSON.parse(line) as ActivityEvent);
}

const ACTIVITY_PATH = 'spwn-home/activity.jsonl';

describe('activity event emissions', () => {
    test('agent creation emits an agent.created event', async () => {
        // Given - a single agent creation
        await using result = await isolated().exec('agent create neo');

        // Then - one agent.created entry carries the full actor/verb/target shape
        expect(result.exitCode).toBe(0);
        const created = parseActivity(result.file(ACTIVITY_PATH).content).filter(
            (e) => e.type === 'agent.created',
        );
        expect(created).toHaveLength(1);
        const event = required(created[0], 'the agent.created event');
        expect(event.actor).toBe('user');
        expect(event.verb).toBe('created');
        expect(event.target).toBe('neo');
        expect(event.agent_id).toBe('neo');
        expect(event.phrase).toBe('You created neo');
    });

    test('agent deletion emits an agent.deleted event', async () => {
        // Given - an agent created then removed
        await using result = await isolated().exec(['agent create neo', 'agent rm neo']);

        // Then - one agent.deleted entry records the removal
        expect(result.exitCode).toBe(0);
        const deleted = parseActivity(result.file(ACTIVITY_PATH).content).filter(
            (e) => e.type === 'agent.deleted',
        );
        expect(deleted).toHaveLength(1);
        const event = required(deleted[0], 'the agent.deleted event');
        expect(event.verb).toBe('deleted');
        expect(event.target).toBe('neo');
        expect(event.phrase).toBe('neo was deleted');
    });

    test('agent fork emits an agent.forked event', async () => {
        // Given - an agent created then forked
        await using result = await isolated().exec(['agent create neo', 'agent fork neo trinity']);

        // Then - one agent.forked entry records the source in metadata
        expect(result.exitCode).toBe(0);
        const forked = parseActivity(result.file(ACTIVITY_PATH).content).filter(
            (e) => e.type === 'agent.forked',
        );
        expect(forked).toHaveLength(1);
        const event = required(forked[0], 'the agent.forked event');
        expect(event.verb).toBe('forked');
        expect(event.target).toBe('trinity');
        expect(event.agent_id).toBe('trinity');
        expect(event.phrase).toBe('trinity forked from neo');
        expect(event.metadata).toBeDefined();
        expect(event.metadata).toHaveProperty('source', 'neo');
    });

    test('agent sleep emits an agent.slept event', async () => {
        // Given - an agent created then put to sleep
        await using result = await isolated().exec(['agent create neo', 'agent sleep neo']);

        // Then - one agent.slept entry is actored by the agent itself
        expect(result.exitCode).toBe(0);
        const slept = parseActivity(result.file(ACTIVITY_PATH).content).filter(
            (e) => e.type === 'agent.slept',
        );
        expect(slept).toHaveLength(1);
        const event = required(slept[0], 'the agent.slept event');
        expect(event.actor).toBe('neo');
        expect(event.verb).toBe('slept');
        expect(event.agent_id).toBe('neo');
    });

    test('event has an id, timestamp, and required fields', async () => {
        // Given - a single creation
        await using result = await isolated().exec('agent create neo');

        // Then - the first entry carries a non-trivial id, ISO timestamp, and core fields
        expect(result.exitCode).toBe(0);
        const events = parseActivity(result.file(ACTIVITY_PATH).content);
        expect(events.length).toBeGreaterThan(0);
        const event = required(events[0], 'the first activity event');
        expect(event.id.length).toBeGreaterThan(10);
        expect(event.timestamp).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
        expect(event.type.length).toBeGreaterThan(0);
        expect(event.actor.length).toBeGreaterThan(0);
        expect(event.verb.length).toBeGreaterThan(0);
        expect(event.phrase.length).toBeGreaterThan(0);
    });

    test('events are appended in chronological order', async () => {
        // Given - three agents created in sequence
        await using result = await isolated().exec([
            'agent create a',
            'agent create b',
            'agent create c',
        ]);

        // Then - the three creation timestamps are non-decreasing
        expect(result.exitCode).toBe(0);
        const creations = parseActivity(result.file(ACTIVITY_PATH).content).filter(
            (e) => e.type === 'agent.created',
        );
        expect(creations).toHaveLength(3);
        const emittedAt = creations.map((e) => new Date(e.timestamp).getTime());
        expect(emittedAt.toSorted((a, b) => a - b)).toStrictEqual(emittedAt);
    });

    test('event ids are unique', async () => {
        // Given - five agents created in one chain
        await using result = await isolated().exec([
            'agent create agent-0',
            'agent create agent-1',
            'agent create agent-2',
            'agent create agent-3',
            'agent create agent-4',
        ]);

        // Then - every emitted id is distinct
        expect(result.exitCode).toBe(0);
        const events = parseActivity(result.file(ACTIVITY_PATH).content);
        const ids = new Set(events.map((e) => e.id));
        expect(ids.size).toBe(events.length);
    });
});
