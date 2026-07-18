import { execSync } from 'node:child_process';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn agent export` / `spwn agent import`.
 *
 * Export writes `<name>.tar.gz` into the current working directory,
 * which for each spec is a fresh temp project — so the archive lands
 * next to `spwn-home/`. The framework reads file text, not archive
 * entries, so the "contents" spec keeps a raw `tar tzf` over
 * `result.filesystem.cwd` to inspect the tarball listing (genuine test
 * plumbing not expressible via the file accessor). The runner is
 * docker-aware, so every result binds with `await using` (rule B5).
 */

// Isolated global-mode home so agent state never leaks between specs.
const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('agent export', () => {
    test('export creates a tar.gz next to the project', async () => {
        // Given - a created agent exported to an archive
        await using result = await isolated().exec(['agent create neo', 'agent export neo']);

        // Then - the archive is on disk (scalpel: banner presence probe)
        expect(result.exitCode).toBe(0);
        expect(result.file('neo.tar.gz').exists).toBe(true);
        expect(result.stderr).toContain('Exported');
    });

    test('exported archive contains all mind layers', async () => {
        // Given - a created agent exported to an archive
        await using result = await isolated().exec(['agent create neo', 'agent export neo']);

        // Then - the tarball listing carries the Soul plus the two Mind layer dirs (scalpel: tar listing probe, not framework-expressible)
        expect(result.exitCode).toBe(0);
        const tarPath = join(result.filesystem.cwd, 'neo.tar.gz');
        const listing = execSync(`tar tzf ${tarPath}`, { encoding: 'utf8' });
        expect(listing).toContain('SOUL.md');
        expect(listing).toMatch(/(?:^|\n)playbooks(?:\/|\n|$)/);
        expect(listing).toMatch(/(?:^|\n)journal(?:\/|\n|$)/);
    });

    test('export --exclude still succeeds', async () => {
        // Given - a created agent exported while excluding some layers
        await using result = await isolated().exec([
            'agent create neo',
            'agent export neo --exclude journal,sessions',
        ]);

        // Then - the archive is still written (scalpel: banner presence probe)
        expect(result.exitCode).toBe(0);
        expect(result.file('neo.tar.gz').exists).toBe(true);
        expect(result.stderr).toContain('Exported');
    });

    test('export on a missing agent errors cleanly', async () => {
        // Given - export against an agent that was never created
        await using result = await isolated().exec('agent export ghost');

        // Then - exit 1 with the canonical export-failed banner and no panic
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('export-missing-agent.txt');
        expect(result.stderr).not.toContain('panic:');
    });

    test('import --as renames the agent on import', async () => {
        // Given - a fresh export of neo, removed, then imported under a new name
        await using result = await isolated().exec([
            'agent create neo',
            'agent export neo',
            'agent rm neo',
            'agent import neo.tar.gz --as neo-copy',
        ]);

        // Then - the restored dir carries the new name and the old name is gone
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/agents/neo-copy/SOUL.md').exists).toBe(true);
        expect(result.file('spwn-home/agents/neo/SOUL.md').exists).toBe(false);
    });

    test('import restores an agent from its own export', async () => {
        // Given - a create -> export -> rm -> import round-trip
        await using result = await isolated().exec([
            'agent create neo',
            'agent export neo',
            'agent rm neo',
            'agent import neo.tar.gz',
        ]);

        // Then - the agent is restored on disk (scalpel: banner presence probe)
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.stderr).toContain('Imported agent');
    });
});
