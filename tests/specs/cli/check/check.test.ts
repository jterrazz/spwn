import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn check` — full-surface goldens for the deterministic reports plus scalpel
 * probes for the paths whose output is inherently dynamic. The temp cwd is
 * covered by the {{workdir}} token. The runner is docker-aware, so every result
 * binds with `await using` even though check spawns no containers (rule B5).
 */

test('valid project prints a clean success report', async () => {
    // Given - the frozen single-agent fixture (one agent, one world)
    await using result = await cli.fixture('$FIXTURES/single-agent/').exec('check');

    // Then - exits zero with the canonical "Project is valid" banner
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toMatch('valid-project.txt');
});

test('--help prints the check command usage', async () => {
    // Given - any cwd; --help is resolved before the project walk
    await using result = await cli.fixture('$FIXTURES/empty/').exec('check --help');

    // Then - cobra emits the usage block for the check subcommand
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toMatch('help.txt');
});

test('freshly initialised project passes check cleanly', async () => {
    // Given - an empty project initialised then checked in one chain
    await using result = await cli
        .fixture('$FIXTURES/empty/')
        .exec(['init --name init-check-test', 'check']);

    // Then - the check leg passes (the combined output carries the dynamic init banner, so this is a presence probe)
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('Project is valid');
});

test('flags an agent that references a non-existent built-in tool', async () => {
    // Given - check-invalid-tool has tools: ["spwn:nonexistent"]
    await using result = await cli.fixture('$FIXTURES/check-invalid-tool/').exec('check');

    // Then - exits non-zero and lists the built-ins the user can pick from
    expect(result.exitCode).toBe(1);
    expect(result.stdout).toMatch('invalid-tool-ref.txt');
});

test('flags a remote-registry tool reference as unsupported', async () => {
    // Given - check-registry-tool has dependencies: ["github:jterrazz/foo"]
    await using result = await cli.fixture('$FIXTURES/check-registry-tool/').exec('check');

    // Then - exits non-zero with the "remote registries not yet supported" rule
    expect(result.exitCode).toBe(1);
    expect(result.stdout).toMatch('registry-not-supported.txt');
});

test('overlay flags the one-agent-one-world rule', async () => {
    // Given - single-agent base + an overlay adding a second world claiming the same neo agent
    await using result = await cli
        .fixture('$FIXTURES/single-agent/')
        .fixture('two-worlds-same-agent/')
        .exec('check');

    // Then - check fails and the second world is genuinely present on disk
    expect(result.exitCode).toBe(1);
    expect(result.file('spwn.yaml').content).toContain('duplicate:');
    expect(result.stdout).toContain('already deployed by world "duplicate"');
    expect(result.stdout).toContain('spwn.yaml#worlds.neo.agents');
});

test('ignores stray files under agent directories', async () => {
    // Given - single-agent base + a stray markdown under spwn/agents/neo/ (not a validated skill path)
    await using result = await cli
        .fixture('$FIXTURES/single-agent/')
        .fixture('stray-skill/')
        .exec('check');

    // Then - check still passes; the stray file is invisible to the validator
    expect(result.exitCode).toBe(0);
    expect(result.stdout).not.toContain('naked.md');
    expect(result.stdout).not.toContain('missing YAML frontmatter');
});

test('emits a JSON report for a valid project', async () => {
    // Given - the frozen single-agent fixture
    await using result = await cli.fixture('$FIXTURES/single-agent/').exec('check --json');

    // Then - exits zero and emits the canonical JSON envelope
    expect(result.exitCode).toBe(0);
    expect(result.json).toMatch('valid.json');
});

test('emits a JSON report listing rule violations', async () => {
    // Given - check-invalid-tool references a nonexistent built-in
    await using result = await cli.fixture('$FIXTURES/check-invalid-tool/').exec('check --json');

    // Then - exits non-zero and the issue list is structurally stable
    expect(result.exitCode).toBe(1);
    expect(result.json).toMatch('invalid-tool.json');
});

test('--deep passes on a valid project', async () => {
    // Given - the single-agent fixture has a real AGENTS.md so the compile pass is clean
    await using result = await cli.fixture('$FIXTURES/single-agent/').exec('check --deep');

    // Then - exits zero with the clean success banner
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toMatch('valid-project.txt');
});

test('--deep catches an empty AGENTS.md that shallow check misses', async () => {
    // Given - an overlay wipes AGENTS.md to zero bytes; the manifest rule only checks existence
    await using shallow = await cli
        .fixture('$FIXTURES/single-agent/')
        .fixture('empty-agents-md/')
        .exec('check');
    await using deep = await cli
        .fixture('$FIXTURES/single-agent/')
        .fixture('empty-agents-md/')
        .exec('check --deep');

    // Then - shallow passes but deep exits non-zero surfacing the empty prompt
    expect(shallow.exitCode).toBe(0);
    expect(deep.exitCode).toBe(1);
    expect(deep.stdout).toContain('agent prompt is missing or empty');
    expect(deep.stdout).toContain('spwn/agents/neo/AGENTS.md');
});

test('--deep --json tags compile issues with source=compile', async () => {
    // Given - an overlay with an empty AGENTS.md, checked deep as JSON
    await using result = await cli
        .fixture('$FIXTURES/single-agent/')
        .fixture('empty-agents-md/')
        .exec('check --deep --json');

    // Then - the report flags a compile-sourced issue (scalpel: the compile issue set is dynamic)
    expect(result.exitCode).toBe(1);
    const report = result.json.value as {
        issues: Array<{ level: string; message: string; source?: string }>;
        summary: { errors: number };
        valid: boolean;
    };
    expect(report.valid).toBe(false);
    const compileIssues = report.issues.filter((issue) => issue.source === 'compile');
    expect(compileIssues.length).toBeGreaterThan(0);
    expect(compileIssues[0].message).toContain('agent prompt');
});

test('errors when run outside a spwn project', async () => {
    // Given - the empty fixture has no spwn.yaml anywhere up the tree
    await using result = await cli.fixture('$FIXTURES/empty/').exec('check');

    // Then - exits non-zero and nudges the user at spwn init (scalpel: stderr lowercased for a case-insensitive probe)
    expect(result.exitCode).toBe(1);
    const stderr = result.stderr.text.toLowerCase();
    expect(stderr).toContain('spwn init');
    expect(stderr).toContain('spwn.yaml');
});
