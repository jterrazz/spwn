import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * E2E coverage for `spwn compile` — the disk materialisation path
 * introduced in Phase 2 of the compiler refactor. Each test pins one
 * flag or error path: project resolution, --dry-run, --out, --agent,
 * --json, --runtime fallback, and the "no spwn.yaml" diagnostic.
 *
 * These exercise the CLI wiring only — the renderer itself is covered
 * by Go golden-fixture tests in packages/compile/runtimes/claudecode/.
 */
describe('spwn compile', () => {
    test('errors when run outside a spwn project', async () => {
        const result = await spec('compile no project').project('empty').exec('compile').run();

        expect(result.exitCode).toBe(1);
        const stderr = result.stderr.text.toLowerCase();
        expect(stderr).toContain('spwn init');
        expect(stderr).toContain('spwn.yaml');
    });

    test('writes a Tree to ./dist on a minimal project', async () => {
        const result = await spec('compile default out')
            .project('docker-pilot')
            .exec('compile')
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('dist/agents/neo/CLAUDE.md').exists).toBe(true);
        expect(result.file('dist/world/physics.md').exists).toBe(true);
        expect(result.file('dist/world/faculties.md').exists).toBe(true);
        expect(result.file('dist/world/AGENTS.md').exists).toBe(true);
    });

    test('--dry-run prints paths without touching disk', async () => {
        const result = await spec('compile dry run')
            .project('docker-pilot')
            .exec('compile --dry-run')
            .run();

        expect(result.exitCode).toBe(0);
        result.stdout.toContain('agents/neo/CLAUDE.md');
        result.stdout.toContain('world/physics.md');
        expect(result.file('dist').exists).toBe(false);
    });

    test('--out writes to a custom location', async () => {
        const result = await spec('compile out custom')
            .project('docker-pilot')
            .exec('compile --out build/preview')
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('build/preview/agents/neo/CLAUDE.md').exists).toBe(true);
        expect(result.file('build/preview/world/physics.md').exists).toBe(true);
        expect(result.file('dist').exists).toBe(false);
    });

    test('--json emits a machine-readable report', async () => {
        const result = await spec('compile json')
            .project('docker-pilot')
            .exec('compile --json')
            .run();

        expect(result.exitCode).toBe(0);
        const report = result.json.value as {
            fileCount: number;
            outDir: string;
            paths: string[];
            runtime: string;
        };
        expect(report.runtime).toBe('claude-code');
        expect(report.fileCount).toBeGreaterThan(0);
        expect(Array.isArray(report.paths)).toBe(true);
        expect(report.paths).toContain('agents/neo/CLAUDE.md');
        expect(report.paths).toContain('world/physics.md');
        expect(report.fileCount).toBe(report.paths.length);
    });

    test('--agent filters the Tree to one agent in a colony', async () => {
        const result = await spec('compile agent filter')
            .project('docker-pilot')
            .seed('agent/morpheus')
            .seed('spwn.yaml/colony.yaml')
            .exec('compile --agent neo --dry-run')
            .run();

        expect(result.exitCode).toBe(0);
        result.stdout.toContain('agents/neo/CLAUDE.md');
        const text = result.stdout.text;
        expect(text).not.toContain('agents/morpheus');
        expect(text).not.toContain('world/physics.md');
    });

    test('--runtime <bogus> errors with a hint about known runtimes', async () => {
        const result = await spec('compile bad runtime')
            .project('docker-pilot')
            .exec('compile --runtime codex')
            .run();

        expect(result.exitCode).toBe(1);
        const stderr = result.stderr.text.toLowerCase();
        expect(stderr).toContain('codex');
        expect(stderr).toContain('claude-code');
    });
});
