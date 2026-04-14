import { spawnSync } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { afterEach, beforeEach, describe, expect, test } from 'vitest';

import { createSpwnHome } from '../../setup/helpers.js';

const SPWN_BIN = resolve(import.meta.dirname, '../../../bin/spwn');

interface ActivityEvent {
    id: string;
    timestamp: string;
    type: string;
    actor: string;
    verb: string;
    target?: string;
    phrase: string;
    world_id?: string;
    agent_id?: string;
    metadata?: Record<string, unknown>;
}

function runSpwn(args: string[], home: string): { exitCode: number; output: string } {
    const result = spawnSync(SPWN_BIN, args, {
        encoding: 'utf8',
        env: { ...process.env, SPWN_HOME: home, INIT_CWD: undefined } as NodeJS.ProcessEnv,
        timeout: 30_000,
    });
    return {
        exitCode: result.status ?? 1,
        output: (result.stdout ?? '') + (result.stderr ?? ''),
    };
}

function readActivityLog(home: string): ActivityEvent[] {
    const path = join(home, 'activity.jsonl');
    if (!existsSync(path)) {
        return [];
    }
    const content = readFileSync(path, 'utf8');
    return content
        .split('\n')
        .filter((l) => l.trim())
        .map((l) => JSON.parse(l) as ActivityEvent);
}

function eventsOfType(events: ActivityEvent[], type: string): ActivityEvent[] {
    return events.filter((e) => e.type === type);
}

describe('activity event emissions', () => {
    let home: string;

    beforeEach(() => {
        home = createSpwnHome();
    });

    afterEach(() => {
        spawnSync('rm', ['-rf', home], { timeout: 5000 });
    });

    test('agent creation emits agent.created event', () => {
        // WHEN
        const result = runSpwn(['agent', 'new', 'neo'], home);
        expect(result.exitCode).toBe(0);

        // THEN
        const events = readActivityLog(home);
        const created = eventsOfType(events, 'agent.created');
        expect(created.length).toBe(1);
        expect(created[0].actor).toBe('user');
        expect(created[0].verb).toBe('created');
        expect(created[0].target).toBe('neo');
        expect(created[0].agent_id).toBe('neo');
        expect(created[0].phrase).toBe('You created neo');
    });

    test('agent deletion emits agent.deleted event', () => {
        // GIVEN
        runSpwn(['agent', 'new', 'neo'], home);

        // WHEN
        const result = runSpwn(['agent', 'rm', 'neo'], home);
        expect(result.exitCode).toBe(0);

        // THEN
        const events = readActivityLog(home);
        const deleted = eventsOfType(events, 'agent.deleted');
        expect(deleted.length).toBe(1);
        expect(deleted[0].verb).toBe('deleted');
        expect(deleted[0].target).toBe('neo');
        expect(deleted[0].phrase).toBe('neo was deleted');
    });

    test('agent fork emits agent.forked event', () => {
        // GIVEN
        runSpwn(['agent', 'new', 'neo'], home);

        // WHEN
        const result = runSpwn(['agent', 'fork', 'neo', 'trinity'], home);
        expect(result.exitCode).toBe(0);

        // THEN
        const events = readActivityLog(home);
        const forked = eventsOfType(events, 'agent.forked');
        expect(forked.length).toBe(1);
        expect(forked[0].verb).toBe('forked');
        expect(forked[0].target).toBe('trinity');
        expect(forked[0].agent_id).toBe('trinity');
        expect(forked[0].phrase).toBe('trinity forked from neo');
        expect(forked[0].metadata).toBeDefined();
        expect(forked[0].metadata).toHaveProperty('source', 'neo');
    });

    test('agent dream emits agent.dreamed event when it runs', async () => {
        // GIVEN - an agent with at least one journal entry
        runSpwn(['agent', 'new', 'neo'], home);
        // Seed a fake journal entry so dream doesn't skip
        const journalDir = join(home, 'agents', 'neo', 'journal');
        const { mkdirSync, writeFileSync } = await import('node:fs');
        mkdirSync(journalDir, { recursive: true });
        writeFileSync(
            join(journalDir, '2026-04-04_120000_w-test-00001.md'),
            `# Session Journal\n\n- **World:** w-test-00001\n- **Outcome:** completed\n- **Exit Code:** 0\n- **Duration:** 1m\n- **Started:** 2026-04-04T11:59:00Z\n- **Ended:** 2026-04-04T12:00:00Z\n`,
        );

        // WHEN
        const result = runSpwn(['agent', 'dream', 'neo'], home);
        expect(result.exitCode).toBe(0);

        // THEN
        const events = readActivityLog(home);
        const dreamed = eventsOfType(events, 'agent.dreamed');
        expect(dreamed.length).toBe(1);
        expect(dreamed[0].actor).toBe('neo');
        expect(dreamed[0].verb).toBe('dreamed');
        expect(dreamed[0].agent_id).toBe('neo');
    });

    test('agent sleep emits agent.slept event', () => {
        // GIVEN
        runSpwn(['agent', 'new', 'neo'], home);

        // WHEN
        const result = runSpwn(['agent', 'sleep', 'neo'], home);
        expect(result.exitCode).toBe(0);

        // THEN
        const events = readActivityLog(home);
        const slept = eventsOfType(events, 'agent.slept');
        expect(slept.length).toBe(1);
        expect(slept[0].actor).toBe('neo');
        expect(slept[0].verb).toBe('slept');
        expect(slept[0].agent_id).toBe('neo');
    });

    test('event has ID, timestamp, and required fields', () => {
        runSpwn(['agent', 'new', 'neo'], home);

        const events = readActivityLog(home);
        expect(events.length).toBeGreaterThan(0);
        const e = events[0];

        expect(e.id).toBeTruthy();
        expect(e.id.length).toBeGreaterThan(10); // ID is hex 24 chars
        expect(e.timestamp).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
        expect(e.type).toBeTruthy();
        expect(e.actor).toBeTruthy();
        expect(e.verb).toBeTruthy();
        expect(e.phrase).toBeTruthy();
    });

    test('events are appended in chronological order', () => {
        // GIVEN - sequential operations
        runSpwn(['agent', 'new', 'a'], home);
        runSpwn(['agent', 'new', 'b'], home);
        runSpwn(['agent', 'new', 'c'], home);

        // THEN - events in file are in chronological order (oldest first)
        const events = readActivityLog(home);
        const creations = eventsOfType(events, 'agent.created');
        expect(creations.length).toBe(3);

        for (let i = 1; i < creations.length; i++) {
            const prev = new Date(creations[i - 1].timestamp).getTime();
            const curr = new Date(creations[i].timestamp).getTime();
            expect(curr).toBeGreaterThanOrEqual(prev);
        }
    });

    test('event IDs are unique', () => {
        // GIVEN - multiple events
        for (let i = 0; i < 5; i++) {
            runSpwn(['agent', 'new', `agent-${i}`], home);
        }

        // THEN - all event IDs are unique
        const events = readActivityLog(home);
        const ids = new Set(events.map((e) => e.id));
        expect(ids.size).toBe(events.length);
    });

    test('activity.jsonl file is created on first event', () => {
        // GIVEN - no file exists
        const path = join(home, 'activity.jsonl');
        expect(existsSync(path)).toBe(false);

        // WHEN - an event is emitted
        runSpwn(['agent', 'new', 'neo'], home);

        // THEN - file is created
        expect(existsSync(path)).toBe(true);
    });

    test('each line is valid JSON', () => {
        // GIVEN - several events
        runSpwn(['agent', 'new', 'neo'], home);
        runSpwn(['agent', 'new', 'morpheus'], home);
        runSpwn(['agent', 'fork', 'neo', 'trinity'], home);

        // THEN - file contains only valid JSON lines
        const path = join(home, 'activity.jsonl');
        const content = readFileSync(path, 'utf8');
        const lines = content.split('\n').filter((l) => l.trim());
        expect(lines.length).toBeGreaterThan(0);
        for (const line of lines) {
            expect(() => JSON.parse(line)).not.toThrow();
        }
    });
});
