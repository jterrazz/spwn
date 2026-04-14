import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Graceful shutdown under the docker() spec mode.
 *
 * The legacy tests exercised:
 *   - `spwn down <world-id>` with explicit ids
 *   - `spwn down --all`
 *   - journal entries written on shutdown
 *   - `spwn upgrade --help`
 *
 * In project mode the world-id path is no longer the primary interface:
 * bare `spwn down` tears down every world declared in spwn.yaml (which
 * maps to what the legacy `--all` flag did). The explicit `--all` flag
 * no longer exists in the project-mode CLI surface.
 *
 * Restored (was wrongly dropped):
 *   - Journal-entry-on-destroy assertions. The destroy path in
 *     `packages/world/internal/architect/destroy.go` does call
 *     `mind.AppendJournal` for every deployed agent. The earlier
 *     docstring claiming no code path writes journal entries on
 *     teardown was incorrect; the "journal on destroy" test below
 *     now locks that contract down.
 *
 * Preserved from legacy:
 *   - `down` exits cleanly and removes the container from docker
 *   - `down` followed by a second `down` doesn't panic
 *   - `spwn upgrade --help` documents the release-download behaviour and
 *     that running worlds are stopped before swapping the binary
 *
 * Augmented:
 *   - Checks the destroy banners explicitly on stderr
 *   - Confirms `world list --json` reports no running worlds post-down
 */
describe('graceful shutdown', () => {
    test('down on a running project world removes the container cleanly', async () => {
        await using result = await spec('graceful down')
            .project('docker-pilot')
            .exec(['up', 'down'])
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Destroyed');
        result.stderr.toContain('project world(s) destroyed');

        // Container is gone from docker
        expect(result.container('neo').exists).toBe(false);
    });

    test('world list --json reports no running worlds after down', async () => {
        await using downed = await spec('graceful down then list up')
            .project('docker-pilot')
            .exec(['up', 'down'])
            .run();

        expect(downed.exitCode).toBe(0);

        await using list = await spec('graceful down then list')
            .project('docker-pilot')
            .exec('world list --json')
            .run();

        expect(list.exitCode).toBe(0);
        const report = list.json.value as {
            mode: string;
            worlds: Array<{ name: string; status: string }>;
        };
        expect(report.mode).toBe('project');
        expect(report.worlds.every((w) => w.status !== 'running')).toBe(true);
    });

    test('double down is idempotent (no panic, no stack trace)', async () => {
        await using result = await spec('graceful double down')
            .project('docker-pilot')
            .exec(['up', 'down', 'down'])
            .run();

        // The second `down` may exit non-zero (no worlds to stop), but
        // It must not crash or produce a stack trace.
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');
        expect(combined).not.toContain('FATAL');
        expect(combined).not.toContain('TypeError');

        // Container is still gone after the double-down
        expect(result.container('neo').exists).toBe(false);
    });

    test('down with no running worlds succeeds gracefully', async () => {
        await using result = await spec('graceful down empty')
            .project('docker-pilot')
            .exec('down')
            .run();

        // Nothing to destroy — should not panic regardless of exit code
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');
        expect(combined).not.toContain('FATAL');
        // And the explicit zero-count banner fires so users see that
        // Spwn recognised there was nothing to stop (vs. a silent exit).
        expect(combined).toContain('No project worlds were running');
    });

    test('down appends a journal entry to the destroyed agent', async () => {
        // Given - a running project world with neo,
        // When - we up then down,
        // Then - the architect's destroy path must append a journal entry
        // Under spwn/agents/neo/journal/ (see
        // Packages/world/internal/architect/destroy.go:54 calling
        // Mind.AppendJournal). This test guards that contract.
        await using result = await spec('journal on destroy')
            .project('docker-pilot')
            .exec(['up', 'down'])
            .run();

        expect(result.exitCode).toBe(0);

        const journalDir = result.directory('spwn/agents/neo/journal');
        const entries = await journalDir.files();
        expect(entries.length).toBeGreaterThanOrEqual(1);
    });

    test('upgrade --help documents graceful world shutdown', async () => {
        await using result = await spec('graceful upgrade help')
            .project('docker-pilot')
            .exec('upgrade --help')
            .run();

        expect(result.exitCode).toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).toMatch(/[Dd]ownloads.*spwn release/);
        expect(combined).toMatch(/[Rr]unning worlds are stopped/);
    });
});
