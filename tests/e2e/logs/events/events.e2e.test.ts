import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Activity event emissions — asserts that CLI operations append well-
 * formed entries to `activity.jsonl`. Each test uses an isolated
 * `$WORKDIR/spwn-home` so its activity log starts empty, then parses
 * the file directly after the chained commands run.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

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
        .filter((l) => l.trim())
        .map((l) => JSON.parse(l) as ActivityEvent);
}

const ACTIVITY_PATH = 'spwn-home/activity.jsonl';

describe('activity event emissions', () => {
    test('agent creation emits agent.created event', async () => {
        const result = await isolated('emit agent.created').exec('agent create neo').run();
        expect(result.exitCode).toBe(0);

        const events = parseActivity(result.file(ACTIVITY_PATH).content);
        const created = events.filter((e) => e.type === 'agent.created');
        expect(created).toHaveLength(1);
        expect(created[0].actor).toBe('user');
        expect(created[0].verb).toBe('created');
        expect(created[0].target).toBe('neo');
        expect(created[0].agent_id).toBe('neo');
        expect(created[0].phrase).toBe('You created neo');
    });

    test('agent deletion emits agent.deleted event', async () => {
        const result = await isolated('emit agent.deleted')
            .exec(['agent create neo', 'agent rm neo'])
            .run();
        expect(result.exitCode).toBe(0);

        const events = parseActivity(result.file(ACTIVITY_PATH).content);
        const deleted = events.filter((e) => e.type === 'agent.deleted');
        expect(deleted).toHaveLength(1);
        expect(deleted[0].verb).toBe('deleted');
        expect(deleted[0].target).toBe('neo');
        expect(deleted[0].phrase).toBe('neo was deleted');
    });

    test('agent fork emits agent.forked event', async () => {
        const result = await isolated('emit agent.forked')
            .exec(['agent create neo', 'agent fork neo trinity'])
            .run();
        expect(result.exitCode).toBe(0);

        const events = parseActivity(result.file(ACTIVITY_PATH).content);
        const forked = events.filter((e) => e.type === 'agent.forked');
        expect(forked).toHaveLength(1);
        expect(forked[0].verb).toBe('forked');
        expect(forked[0].target).toBe('trinity');
        expect(forked[0].agent_id).toBe('trinity');
        expect(forked[0].phrase).toBe('trinity forked from neo');
        expect(forked[0].metadata).toBeDefined();
        expect(forked[0].metadata).toHaveProperty('source', 'neo');
    });

    test('agent sleep emits agent.slept event', async () => {
        const result = await isolated('emit agent.slept')
            .exec(['agent create neo', 'agent sleep neo'])
            .run();
        expect(result.exitCode).toBe(0);

        const events = parseActivity(result.file(ACTIVITY_PATH).content);
        const slept = events.filter((e) => e.type === 'agent.slept');
        expect(slept).toHaveLength(1);
        expect(slept[0].actor).toBe('neo');
        expect(slept[0].verb).toBe('slept');
        expect(slept[0].agent_id).toBe('neo');
    });

    test('event has ID, timestamp, and required fields', async () => {
        const result = await isolated('event shape').exec('agent create neo').run();
        expect(result.exitCode).toBe(0);

        const events = parseActivity(result.file(ACTIVITY_PATH).content);
        expect(events.length).toBeGreaterThan(0);
        const e = events[0];

        expect(e.id).toBeTruthy();
        // ID is hex (24 chars in current impl)
        expect(e.id.length).toBeGreaterThan(10);
        expect(e.timestamp).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
        expect(e.type).toBeTruthy();
        expect(e.actor).toBeTruthy();
        expect(e.verb).toBeTruthy();
        expect(e.phrase).toBeTruthy();
    });

    test('events are appended in chronological order', async () => {
        const result = await isolated('chronological')
            .exec(['agent create a', 'agent create b', 'agent create c'])
            .run();
        expect(result.exitCode).toBe(0);

        const events = parseActivity(result.file(ACTIVITY_PATH).content);
        const creations = events.filter((e) => e.type === 'agent.created');
        expect(creations).toHaveLength(3);

        for (let i = 1; i < creations.length; i++) {
            const prev = new Date(creations[i - 1].timestamp).getTime();
            const curr = new Date(creations[i].timestamp).getTime();
            expect(curr).toBeGreaterThanOrEqual(prev);
        }
    });

    test('event IDs are unique', async () => {
        const result = await isolated('unique ids')
            .exec([
                'agent create agent-0',
                'agent create agent-1',
                'agent create agent-2',
                'agent create agent-3',
                'agent create agent-4',
            ])
            .run();
        expect(result.exitCode).toBe(0);

        const events = parseActivity(result.file(ACTIVITY_PATH).content);
        const ids = new Set(events.map((e) => e.id));
        expect(ids.size).toBe(events.length);
    });

    test('activity.jsonl file is created on first event', async () => {
        const before = await isolated('before any event').exec('--version').run();
        expect(before.file(ACTIVITY_PATH).exists).toBe(false);

        const after = await isolated('after first event').exec('agent create neo').run();
        expect(after.exitCode).toBe(0);
        expect(after.file(ACTIVITY_PATH).exists).toBe(true);
    });

    test('each line is valid JSON', async () => {
        const result = await isolated('valid json lines')
            .exec(['agent create neo', 'agent create morpheus', 'agent fork neo trinity'])
            .run();
        expect(result.exitCode).toBe(0);

        const content = result.file(ACTIVITY_PATH).content;
        const lines = content.split('\n').filter((l) => l.trim());
        expect(lines.length).toBeGreaterThan(0);
        for (const line of lines) {
            expect(() => JSON.parse(line)).not.toThrow();
        }
    });
});
