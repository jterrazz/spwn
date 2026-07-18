import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn logs` CLI surface. Each spec runs in a fresh temp project with an
 * isolated `$WORKDIR/spwn-home`; activity events land at
 * `spwn-home/activity.jsonl`, so chained `exec([...])` calls share one home
 * across sub-commands. Log lines carry timestamps, so most probes stay
 * scalpels (rule D11(b)); only the deterministic --help output is a golden.
 * The runner is docker-aware, so every result binds with `await using` (B5).
 */

const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('spwn logs cli', () => {
    test('empty log shows a friendly message', async () => {
        // Given - a home with no activity yet
        await using result = await isolated().exec('logs');

        // Then - the empty-state message prints (scalpel: case-insensitive probe)
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toMatch(/No events yet/i);
    });

    test('events appear after agent creation', async () => {
        // Given - an agent created then the log read in one chain
        await using result = await isolated().exec(['agent create neo', 'logs']);

        // Then - the creation event surfaces (scalpel: log lines carry timestamps)
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toMatch(/agent\.created/);
        expect(result.stdout).toContain('You created neo');
    });

    test('events accumulate across multiple operations', async () => {
        // Given - three agents created then the log read
        await using result = await isolated().exec([
            'agent create neo',
            'agent create morpheus',
            'agent create trinity',
            'logs',
        ]);

        // Then - every creation is present (scalpel: timestamped log lines)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('You created neo');
        expect(result.stdout).toContain('You created morpheus');
        expect(result.stdout).toContain('You created trinity');
    });

    test('--limit restricts the number of events', async () => {
        // Given - five creations then a two-event window
        await using result = await isolated().exec([
            'agent create a',
            'agent create b',
            'agent create c',
            'agent create d',
            'agent create e',
            'logs --limit 2',
        ]);

        // Then - only two creation lines render (scalpel: counting timestamped lines)
        expect(result.exitCode).toBe(0);
        const lines = result.stdout.text
            .split('\n')
            .filter((line) => line.includes('agent.created'));
        expect(lines.length).toBe(2);
    });

    test('--type filters by event type', async () => {
        // Given - a create then a remove, filtered to created events
        await using result = await isolated().exec([
            'agent create neo',
            'agent rm neo',
            'logs --type agent.created',
        ]);

        // Then - the creation shows but the deletion is filtered out (scalpel: absence + timestamped lines)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('You created neo');
        expect(result.stdout.text).not.toMatch(/neo was deleted/);
    });

    test('--agent filters by agent name', async () => {
        // Given - two agents created, filtered to neo
        await using result = await isolated().exec([
            'agent create neo',
            'agent create morpheus',
            'logs --agent neo',
        ]);

        // Then - only neo's events remain (scalpel: absence probe)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('You created neo');
        expect(result.stdout).not.toContain('morpheus');
    });

    test('agent logs <name> is a shortcut for --agent', async () => {
        // Given - two agents created, then the per-agent shortcut for neo
        await using result = await isolated().exec([
            'agent create neo',
            'agent create morpheus',
            'agent logs neo',
        ]);

        // Then - only neo's events remain (scalpel: absence probe)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('You created neo');
        expect(result.stdout).not.toContain('morpheus');
    });

    test('output includes a timestamp', async () => {
        // Given - one creation then the log read
        await using result = await isolated().exec(['agent create neo', 'logs']);

        // Then - a HH:MM:SS timestamp is rendered (scalpel: regex probe)
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toMatch(/\d{2}:\d{2}:\d{2}/);
    });

    test('--type with an unknown event type errors', async () => {
        // Given - a bogus --type value
        await using result = await isolated().exec('logs --type garbage');

        // Then - exits non-zero listing the known types (scalpel: error wording)
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toMatch(/unknown event type/i);
        expect(result.stderr.text).toMatch(/agent\.created/);
    });

    test('--world with a name not in spwn.yaml errors', async () => {
        // Given - a single-world project filtered by a world it does not declare
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec('logs --world doesnotexist');

        // Then - exits non-zero naming the unknown world (scalpel: error wording)
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toMatch(/unknown world/i);
    });

    test('--help shows the command description', async () => {
        // Given - the logs help flag
        await using result = await isolated().exec('logs --help');

        // Then - cobra prints the full logs usage block
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('help.txt');
    });
});
