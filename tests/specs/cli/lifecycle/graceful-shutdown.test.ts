import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Graceful shutdown under docker-aware mode. In project mode bare `spwn down`
 * tears down every world declared in spwn.yaml (what the legacy `--all` did).
 * The destroy path appends a journal entry for every deployed agent, and
 * `upgrade --help` documents that running worlds are stopped before the binary
 * swap. Each spawning result binds with `await using` (rule B5).
 */
describe('graceful shutdown', () => {
    test('down on a running project world removes the container cleanly', async () => {
        // Given - a world brought up then torn down
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec(['up', 'down']);

        // Then - the destroy banners fire and the container is gone
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Destroyed');
        expect(result.stderr).toContain('project world(s) destroyed');
        expect(result.container('neo').exists).toBe(false);
    });

    test('world list --json reports no running worlds after down', async () => {
        // Given - a world brought up, torn down, then listed in one chain
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'down', 'world list --json']);

        // Then - no world remains running
        expect(result.exitCode).toBe(0);
        const report = result.json.value as {
            mode: string;
            worlds: Array<{ name: string; status: string }>;
        };
        expect(report.mode).toBe('project');
        expect(report.worlds.every((w) => w.status !== 'running')).toBe(true);
    });

    test('double down is idempotent with no panic or stack trace', async () => {
        // Given - up then two downs in one chain
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'down', 'down']);

        // Then - no crash signals and the container stays gone (scalpel: crash-signal absence)
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
        expect(result.stderr.text).not.toContain('FATAL');
        expect(result.stderr.text).not.toContain('TypeError');
        expect(result.container('neo').exists).toBe(false);
    });

    test('down with no running worlds succeeds gracefully', async () => {
        // Given - a down with nothing to destroy
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('down');

        // Then - no crash and the explicit zero-count banner fires (scalpel: crash absence + banner probe)
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
        expect(result.stderr.text).not.toContain('FATAL');
        expect(result.stderr).toContain('No project worlds were running');
    });

    test('down appends a journal entry to the destroyed agent', async () => {
        // Given - a running world with neo, brought up then down
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec(['up', 'down']);

        // Then - the architect's destroy path wrote at least one journal entry for neo
        expect(result.exitCode).toBe(0);
        const entries = await result.directory('spwn/agents/neo/journal').files();
        expect(entries.length).toBeGreaterThanOrEqual(1);
    });

    test('upgrade --help documents graceful world shutdown', async () => {
        // Given - the upgrade help flag
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('upgrade --help');

        // Then - cobra prints the full upgrade usage block covering the release download and world stop
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('upgrade-help.txt');
    });
});
