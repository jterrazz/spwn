import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn init` — project scaffolding. The legacy --global mode is gone;
 * these tests pin the modern project-mode behaviour:
 *
 *   - bare `init` in an empty dir writes the starter project tree
 *   - re-running in a populated dir errors unless --force is passed
 *   - `init @spwn/matrix` installs a bundled template
 *
 * We pass --name so the stdout banner is stable across runs (otherwise
 * the basename of the temp working directory would leak into the
 * "Initialised spwn project <name>" line).
 */

describe('spwn init', () => {
    test('scaffolds the starter project in an empty directory', async () => {
        // Given - the empty fixture (just a .gitkeep)
        const result = await spec('init empty')
            .project('empty')
            .exec('init --name demo-project')
            .run();

        // Then - exits zero, stderr carries the "Initialised" banner
        // And stdout carries the committed/next summary block.
        expect(result.exitCode).toBe(0);
        await result.stderr.toMatch('init-banner.txt');
        await result.stdout.toMatch('init-output.txt');

        // And the key files are on disk
        expect(result.file('spwn.yaml').exists).toBe(true);
        expect(result.file('spwn.yaml').content).toContain('name: demo-project');
        expect(result.file('spwn/agents/neo/agent.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/neo/core/profile.md').exists).toBe(true);
        expect(result.file('spwn/agents/neo/CLAUDE.md').exists).toBe(true);
        expect(result.file('.gitignore').exists).toBe(true);
        expect(result.file('.gitignore').content).toContain('.spwn');
    });

    test('errors when spwn.yaml already exists', async () => {
        // Given - the single-agent fixture already has spwn.yaml
        const result = await spec('init conflict').project('single-agent').exec('init').run();

        // Then - exits non-zero and points at --force
        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('init-already-exists.txt');
    });

    test('--force overwrites an existing spwn.yaml', async () => {
        // Given - the single-agent fixture with a populated tree
        const result = await spec('init force')
            .project('single-agent')
            .exec('init --force --name forced-project')
            .run();

        // Then - exits zero and rewrites spwn.yaml with the new name.
        // The starter project name is the one we passed via --name,
        // Not the original "single-agent".
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn.yaml').content).toContain('name: forced-project');
    });

    test('init rejects --name values that fail the manifest regex', async () => {
        // Given - an empty dir, a --name with spaces
        // When - running init
        // Then - exit 1, no spwn.yaml written
        const result = await spec('init bad name')
            .project('empty')
            .exec('init --name "Has Spaces"')
            .run();
        expect(result.exitCode).toBe(1);
        expect(result.file('spwn.yaml').exists).toBe(false);
        expect(result.stderr.text).toMatch(/invalid --name/i);
    });

    test('init banner only references files that actually exist', async () => {
        // Given - an empty dir scaffolded via init
        // When - scanning the stdout banner for `spwn/...` paths
        // Then - every printed path resolves to something on disk
        const result = await spec('init banner truth')
            .project('empty')
            .exec('init --name demo-project')
            .run();
        expect(result.exitCode).toBe(0);

        // The old banner promised spwn/worlds/default.yaml which was
        // Never created. Regression guard: no removed-path sneaking
        // Back into the template.
        expect(result.stdout.text).not.toContain('spwn/worlds/default.yaml');
        expect(result.file('spwn/worlds/default.yaml').exists).toBe(false);
    });

    test('init @spwn/matrix installs the bundled template', async () => {
        // Given - an empty dir
        const result = await spec('init matrix').project('empty').exec('init @spwn/matrix').run();

        // Then - the template's starter files land on disk
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/neo/agent.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/neo/core/profile.md').exists).toBe(true);
        // The "Installed template @spwn/matrix" banner lands on stderr.
        await result.stderr.toMatch('init-matrix-banner.txt');
        // And the stdout summary describes what was added.
        const out = result.stdout.text;
        expect(out).toContain('spwn.yaml');
        expect(out).toContain('Worlds added:');
        expect(out).toContain('matrix');
    });
});
