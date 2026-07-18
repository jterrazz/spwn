import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn build --tree-only` — the disk-materialisation path (formerly
 * `spwn compile`). Each test pins one flag or error path: project
 * resolution, --dry-run, --output, --agent, --json, --runtime fallback,
 * and the "no spwn.yaml" diagnostic. These exercise the CLI wiring only;
 * the renderer itself is covered by Go golden tests. The runner is
 * docker-aware, so every result binds with `await using` (rule B5).
 */

test('errors when run outside a spwn project', async () => {
    // Given - the empty fixture has no spwn.yaml anywhere up the tree
    await using result = await cli.fixture('$FIXTURES/empty/').exec('build --tree-only');

    // Then - exits non-zero nudging the user at spwn init (scalpel: stderr lowercased for a case-insensitive probe)
    expect(result.exitCode).toBe(1);
    const stderr = result.stderr.text.toLowerCase();
    expect(stderr).toContain('spwn init');
    expect(stderr).toContain('spwn.yaml');
});

test('writes a Tree to ./dist on a minimal project', async () => {
    // Given - the docker-pilot fixture compiled to the default output
    await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('build --tree-only');

    // Then - each agent CLAUDE.md exists with the world context inlined (no separate role.md / worlds/ tree)
    expect(result.exitCode).toBe(0);
    expect(result.file('dist/agents/neo/CLAUDE.md').exists).toBe(true);
    expect(
        result.file('dist/agents/neo/CLAUDE.md').content,
        'CLAUDE.md missing inlined Role here block',
    ).toContain('## Role here');
});

test('--dry-run prints paths without touching disk', async () => {
    // Given - the docker-pilot fixture compiled in dry-run mode
    await using result = await cli
        .fixture('$FIXTURES/docker-pilot/')
        .exec('build --tree-only --dry-run');

    // Then - the plan is printed but nothing is written to disk
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('agents/neo/CLAUDE.md');
    expect(result.file('dist').exists).toBe(false);
});

test('--output writes to a custom location', async () => {
    // Given - the docker-pilot fixture compiled to a custom output dir
    await using result = await cli
        .fixture('$FIXTURES/docker-pilot/')
        .exec('build --tree-only --output build/preview');

    // Then - the tree lands under the custom dir and default dist is untouched
    expect(result.exitCode).toBe(0);
    expect(result.file('build/preview/agents/neo/CLAUDE.md').exists).toBe(true);
    expect(result.file('dist').exists).toBe(false);
});

test('--json emits a machine-readable report', async () => {
    // Given - the docker-pilot fixture compiled with --json
    await using result = await cli
        .fixture('$FIXTURES/docker-pilot/')
        .exec('build --tree-only --json');

    // Then - the report carries stable fields (scalpel: treeFiles count + outDir are dynamic)
    expect(result.exitCode).toBe(0);
    const report = result.json.value as {
        outDir: string;
        paths: string[];
        runtime: string;
        treeFiles: number;
        treeOnly: boolean;
    };
    expect(report.runtime).toBe('claude-code');
    expect(report.treeOnly).toBe(true);
    expect(report.treeFiles).toBeGreaterThan(0);
    expect(Array.isArray(report.paths)).toBe(true);
    expect(report.paths).toContain('agents/neo/CLAUDE.md');
    expect(report.treeFiles).toBe(report.paths.length);
});

test('codex runtime writes AGENTS.md, .codex config, and native skills', async () => {
    // Given - the codex-pilot fixture compiled with --json
    await using result = await cli
        .fixture('$FIXTURES/codex-pilot/')
        .exec('build --tree-only --json');

    // Then - the codex conventions hold in both the report and on disk (scalpel: path set probes for absence/presence)
    expect(result.exitCode).toBe(0);
    const report = result.json.value as {
        paths: string[];
        runtime: string;
    };
    expect(report.runtime).toBe('codex');
    expect(report.paths).toContain('agents/neo/AGENTS.md');
    expect(report.paths).toContain('agents/neo/.codex/config.toml');
    expect(report.paths).toContain('agents/neo/.agents/skills/focus/SKILL.md');
    expect(report.paths).not.toContain('agents/neo/CLAUDE.md');
    expect(report.paths.some((p) => p.startsWith('agents/neo/.claude/'))).toBe(false);
    expect(report.paths.some((p) => p.startsWith('agents/neo/.codex/skills/'))).toBe(false);
    expect(report.paths.some((p) => p.startsWith('worlds/'))).toBe(false);
    expect(result.file('dist/agents/neo/AGENTS.md').content).toContain('Codex pilot prompt');
    expect(result.file('dist/agents/neo/.codex/config.toml').content).toContain('model = "gpt-5"');
    expect(result.file('dist/agents/neo/.agents/skills/focus/SKILL.md').content).toContain(
        'name: focus',
    );
    expect(result.file('dist/agents/neo/.agents/skills/focus/SKILL.md').content).toContain(
        'Focus Skill',
    );
});

test('--agent filters the Tree to one agent in a colony', async () => {
    // Given - docker-pilot base + a colony overlay adding morpheus and the [morpheus, neo] roster
    await using result = await cli
        .fixture('$FIXTURES/docker-pilot/')
        .fixture('colony/')
        .exec('build --tree-only --agent neo --dry-run');

    // Then - only neo is planned; morpheus is filtered out (scalpel: presence + absence probe on a dry-run plan)
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('agents/neo/CLAUDE.md');
    expect(result.stdout).not.toContain('agents/morpheus');
});

test('--runtime <bogus> errors with a hint about known runtimes', async () => {
    // Given - a runtime name that is never registered (codex is now a real runtime, so it cannot be the bogus input)
    await using result = await cli
        .fixture('$FIXTURES/docker-pilot/')
        .exec('build --tree-only --runtime unknown-runtime');

    // Then - the hint lists the registered runtimes so the user can fix the typo (scalpel: presence probes)
    expect(result.exitCode).toBe(1);
    const stderr = result.stderr.text.toLowerCase();
    expect(stderr).toContain('claude-code');
    expect(stderr).toContain('codex');
});

test('--force re-compile replaces stale files from a filtered run', async () => {
    // Given - a full compile followed by a filtered --force re-compile
    await using result = await cli
        .fixture('$FIXTURES/docker-pilot/')
        .exec(['build --tree-only', 'build --tree-only --agent neo --force']);

    // Then - neo is rewritten and no per-world shared files remain (everything is inlined)
    expect(result.exitCode).toBe(0);
    expect(result.file('dist/agents/neo/CLAUDE.md').exists).toBe(true);
    expect(result.file('dist/world').exists).toBe(false);
});

test('empty AGENTS.md is rejected with a loud error', async () => {
    // Given - docker-pilot base + an overlay wiping neo's AGENTS.md to zero bytes
    await using result = await cli
        .fixture('$FIXTURES/docker-pilot/')
        .fixture('empty-agents-md/')
        .exec('build --tree-only');

    // Then - the compile aborts naming the empty prompt (scalpel: stderr lowercased presence probes)
    expect(result.exitCode).toBe(1);
    const stderr = result.stderr.text.toLowerCase();
    expect(stderr).toContain('agent prompt');
    expect(stderr).toContain('neo');
});

test('--dry-run without --tree-only errors', async () => {
    // Given - a bare build with --dry-run but no --tree-only
    await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('build --dry-run');

    // Then - the CLI rejects the flag combination naming the required flag
    expect(result.exitCode).toBe(1);
    expect(result.stderr).toContain('--tree-only');
});
