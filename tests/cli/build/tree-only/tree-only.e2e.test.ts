import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * E2E coverage for `spwn build --tree-only` - the disk
 * materialisation path formerly known as `spwn compile`. Each test
 * pins one flag or error path: project resolution, --dry-run,
 * --output, --agent, --json, --runtime fallback, and the "no
 * spwn.yaml" diagnostic.
 *
 * These exercise the CLI wiring only - the renderer itself is
 * covered by Go golden-fixture tests in
 * packages/compile/runtimes/claudecode/.
 */
describe('spwn build --tree-only', () => {
    test('errors when run outside a spwn project', async () => {
        const result = await spec('tree-only no project')
            .project('empty')
            .exec('build --tree-only')
            .run();

        expect(result.exitCode).toBe(1);
        const stderr = result.stderr.text.toLowerCase();
        expect(stderr).toContain('spwn init');
        expect(stderr).toContain('spwn.yaml');
    });

    test('writes a Tree to ./dist on a minimal project', async () => {
        const result = await spec('tree-only default out')
            .project('docker-pilot')
            .exec('build --tree-only')
            .run();

        expect(result.exitCode).toBe(0);
        // World-shared context (physics, faculties, roster, AGENTS.md)
        // Is inlined into each agent's CLAUDE.md; no separate files.
        expect(result.file('dist/agents/neo/CLAUDE.md').exists).toBe(true);
        // Per-deployment role.md still lands under worlds/<id>/, where
        // <id> is the world name from spwn.yaml (`neo` in docker-pilot).
        expect(result.file('dist/agents/neo/worlds/neo/role.md').exists).toBe(true);
    });

    test('--dry-run prints paths without touching disk', async () => {
        const result = await spec('tree-only dry run')
            .project('docker-pilot')
            .exec('build --tree-only --dry-run')
            .run();

        expect(result.exitCode).toBe(0);
        result.stdout.toContain('agents/neo/CLAUDE.md');
        expect(result.file('dist').exists).toBe(false);
    });

    test('--output writes to a custom location', async () => {
        const result = await spec('tree-only out custom')
            .project('docker-pilot')
            .exec('build --tree-only --output build/preview')
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('build/preview/agents/neo/CLAUDE.md').exists).toBe(true);
        expect(result.file('dist').exists).toBe(false);
    });

    test('--json emits a machine-readable report', async () => {
        const result = await spec('tree-only json')
            .project('docker-pilot')
            .exec('build --tree-only --json')
            .run();

        expect(result.exitCode).toBe(0);
        const report = result.json.value as {
            treeFiles: number;
            treeOnly: boolean;
            outDir: string;
            paths: string[];
            runtime: string;
        };
        expect(report.runtime).toBe('claude-code');
        expect(report.treeOnly).toBe(true);
        expect(report.treeFiles).toBeGreaterThan(0);
        expect(Array.isArray(report.paths)).toBe(true);
        expect(report.paths).toContain('agents/neo/CLAUDE.md');
        expect(report.treeFiles).toBe(report.paths.length);
    });

    test('--agent filters the Tree to one agent in a colony', async () => {
        const result = await spec('tree-only agent filter')
            .project('docker-pilot')
            .seed('agent/morpheus')
            .seed('spwn.yaml/colony.yaml')
            .exec('build --tree-only --agent neo --dry-run')
            .run();

        expect(result.exitCode).toBe(0);
        result.stdout.toContain('agents/neo/CLAUDE.md');
        const text = result.stdout.text;
        expect(text).not.toContain('agents/morpheus');
    });

    test('--runtime <bogus> errors with a hint about known runtimes', async () => {
        const result = await spec('tree-only bad runtime')
            .project('docker-pilot')
            .exec('build --tree-only --runtime codex')
            .run();

        expect(result.exitCode).toBe(1);
        const stderr = result.stderr.text.toLowerCase();
        expect(stderr).toContain('codex');
        expect(stderr).toContain('claude-code');
    });

    test('--force re-compile replaces stale files from a filtered run', async () => {
        const result = await spec('tree-only force replaces stale')
            .project('docker-pilot')
            .exec(['build --tree-only', 'build --tree-only --agent neo --force'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('dist/agents/neo/CLAUDE.md').exists).toBe(true);
        // No per-world shared files any more — everything is inlined.
        expect(result.file('dist/world').exists).toBe(false);
    });

    test('empty AGENTS.md is rejected with a loud error', async () => {
        const result = await spec('tree-only empty agent md')
            .project('docker-pilot')
            .seed('agent/neo')
            .exec('build --tree-only')
            .run();

        expect(result.exitCode).toBe(1);
        const stderr = result.stderr.text.toLowerCase();
        expect(stderr).toContain('agent prompt');
        expect(stderr).toContain('neo');
    });

    test('--dry-run without --tree-only errors', async () => {
        const result = await spec('tree-only dry-run requires tree-only')
            .project('docker-pilot')
            .exec('build --dry-run')
            .run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toContain('--tree-only');
    });
});
