import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * The two `build --tree-only` specs whose subject is the `--json` report:
 * a `paths` ARRAY judged by membership and by what it must not contain,
 * and a `treeFiles` count cross-checked against its length. A document's
 * `stdout:` is byte-exact text and has no vocabulary for either — and
 * pinning a whole path list byte-for-byte would make every new template
 * file a golden update in a spec that is about the report's shape.
 *
 * Every other tree-only spec — the default write, --dry-run, --output,
 * --agent, --force, the empty prompt and the two refusals — is a document
 * beside this file. The runner is docker-aware, so every result binds
 * with `await using` (rule B5); none of these spawn a container.
 */

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

test('the codex runtime writes AGENTS.md, .codex config and native skills', async () => {
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
