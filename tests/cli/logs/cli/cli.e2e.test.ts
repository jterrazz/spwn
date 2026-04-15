import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn logs` CLI surface.
 *
 * Each spec call gets a fresh temp project with an isolated
 * `$WORKDIR/spwn-home` SPWN_HOME. Activity events land at
 * `spwn-home/activity.jsonl`, so chained `exec([...])` calls share a
 * single home across sub-commands within a test.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('spwn logs CLI', () => {
    test('empty log shows friendly message', async () => {
        const result = await isolated('logs empty').exec('logs').run();
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toMatch(/No events yet/i);
    });

    test('events appear after agent creation', async () => {
        const result = await isolated('logs after create').exec(['agent create neo', 'logs']).run();
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toMatch(/agent\.created/);
        expect(result.stdout.text).toContain('You created neo');
    });

    test('events accumulate across multiple operations', async () => {
        const result = await isolated('logs accumulate')
            .exec(['agent create neo', 'agent create morpheus', 'agent create trinity', 'logs'])
            .run();
        expect(result.exitCode).toBe(0);
        const out = result.stdout.text;
        expect(out).toContain('You created neo');
        expect(out).toContain('You created morpheus');
        expect(out).toContain('You created trinity');
    });

    test('--limit restricts number of events', async () => {
        const result = await isolated('logs limit')
            .exec([
                'agent create a',
                'agent create b',
                'agent create c',
                'agent create d',
                'agent create e',
                'logs --limit 2',
            ])
            .run();
        expect(result.exitCode).toBe(0);
        const lines = result.stdout.text.split('\n').filter((l) => l.includes('agent.created'));
        expect(lines.length).toBe(2);
    });

    test('--type filters by event type', async () => {
        const result = await isolated('logs type filter')
            .exec(['agent create neo', 'agent rm neo', 'logs --type agent.created'])
            .run();
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('You created neo');
        expect(result.stdout.text).not.toMatch(/neo was deleted/);
    });

    test('--agent filters by agent name', async () => {
        const result = await isolated('logs agent filter')
            .exec(['agent create neo', 'agent create morpheus', 'logs --agent neo'])
            .run();
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('You created neo');
        expect(result.stdout.text).not.toContain('morpheus');
    });

    test('agent logs <name> is a shortcut for --agent', async () => {
        const result = await isolated('agent logs shortcut')
            .exec(['agent create neo', 'agent create morpheus', 'agent logs neo'])
            .run();
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('You created neo');
        expect(result.stdout.text).not.toContain('morpheus');
    });

    test('output includes timestamp', async () => {
        const result = await isolated('logs timestamp').exec(['agent create neo', 'logs']).run();
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toMatch(/\d{2}:\d{2}:\d{2}/);
    });

    test('--type with an unknown event type errors', async () => {
        // Given - a bogus --type value
        // When - running logs
        // Then - exit 1 with a list of known types, not silent no-op
        const result = await isolated('logs bad type').exec('logs --type garbage').run();
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toMatch(/unknown event type/i);
        expect(result.stderr.text).toMatch(/agent\.created/);
    });

    test('--world with a name not in spwn.yaml errors', async () => {
        // Given - a project with one declared world (neo)
        // When - filtering by a world that doesn't exist
        // Then - exit 1, with the set of known worlds listed
        const result = await spec('logs bad world')
            .project('single-agent')
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec('logs --world doesnotexist')
            .run();
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toMatch(/unknown world/i);
    });

    test('--help shows command description', async () => {
        const result = await isolated('logs help').exec('logs --help').run();
        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('help.txt');
    });
});
