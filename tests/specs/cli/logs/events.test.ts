import { text } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Activity event emissions — asserts CLI operations append well-formed entries
 * to `activity.jsonl`. Each spec uses an isolated `$WORKDIR/spwn-home` so its
 * log starts empty. The whole-event shape is pinned by a byte-for-byte golden
 * (ids are 24-hex → {{hex}}, timestamps → {{iso8601}}); the per-type specs
 * parse the JSON for targeted field probes. Every result binds with
 * `await using` (rule B5).
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
        expect(created[0].actor).toBe('user');
        expect(created[0].verb).toBe('created');
        expect(created[0].target).toBe('neo');
        expect(created[0].agent_id).toBe('neo');
        expect(created[0].phrase).toBe('You created neo');
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
        expect(deleted[0].verb).toBe('deleted');
        expect(deleted[0].target).toBe('neo');
        expect(deleted[0].phrase).toBe('neo was deleted');
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
        expect(forked[0].verb).toBe('forked');
        expect(forked[0].target).toBe('trinity');
        expect(forked[0].agent_id).toBe('trinity');
        expect(forked[0].phrase).toBe('trinity forked from neo');
        expect(forked[0].metadata).toBeDefined();
        expect(forked[0].metadata).toHaveProperty('source', 'neo');
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
        expect(slept[0].actor).toBe('neo');
        expect(slept[0].verb).toBe('slept');
        expect(slept[0].agent_id).toBe('neo');
    });

    test('event has an id, timestamp, and required fields', async () => {
        // Given - a single creation
        await using result = await isolated().exec('agent create neo');

        // Then - the first entry carries a non-trivial id, ISO timestamp, and core fields
        expect(result.exitCode).toBe(0);
        const events = parseActivity(result.file(ACTIVITY_PATH).content);
        expect(events.length).toBeGreaterThan(0);
        const event = events[0];
        expect(event.id).toBeTruthy();
        expect(event.id.length).toBeGreaterThan(10);
        expect(event.timestamp).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
        expect(event.type).toBeTruthy();
        expect(event.actor).toBeTruthy();
        expect(event.verb).toBeTruthy();
        expect(event.phrase).toBeTruthy();
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
        for (let i = 1; i < creations.length; i += 1) {
            const prev = new Date(creations[i - 1].timestamp).getTime();
            const curr = new Date(creations[i].timestamp).getTime();
            expect(curr).toBeGreaterThanOrEqual(prev);
        }
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

    test('activity.jsonl is created only on the first event', async () => {
        // Given - a no-op invocation writes no log
        await using before = await isolated().exec('--version');
        expect(before.file(ACTIVITY_PATH).exists).toBe(false);

        // Then - the first real event creates the file
        await using after = await isolated().exec('agent create neo');
        expect(after.exitCode).toBe(0);
        expect(after.file(ACTIVITY_PATH).exists).toBe(true);
    });

    test('activity.jsonl matches the whole-event golden', async () => {
        // Given - a mix of create and fork operations
        await using result = await isolated().exec([
            'agent create neo',
            'agent create morpheus',
            'agent fork neo trinity',
        ]);

        // Then - the raw log matches byte-for-byte, ids and timestamps tokenized
        expect(result.exitCode).toBe(0);
        expect(text(result.file(ACTIVITY_PATH).content)).toMatch('activity.jsonl.txt');
    });
});
