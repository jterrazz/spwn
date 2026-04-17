import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Stage 2a of the @jterrazz/test migration: per-feature folder layout
 * with real stdout fixtures.
 *
 * Snapshots live under ./expected/stdout/<name>.txt. Each assertion
 * uses `result.stdout.toMatch('<name>.txt')` — the framework resolves
 * `<name>` against `<this-dir>/expected/stdout/<name>.txt`. Regenerate
 * with `JTERRAZZ_TEST_UPDATE=1 pnpm -C tests exec vitest run cli/check`.
 *
 * Temp-dir paths and ANSI escapes are normalised to `<PROJECT>` by the
 * runner-level `transform` configured in setup/cli.specification.ts, so
 * fixtures stay stable across runs and machines.
 */

describe('spwn check', () => {
    test('valid project prints a clean success report', async () => {
        // Given - the frozen single-agent fixture (one agent, one world)
        const result = await spec('check valid').project('single-agent').exec('check').run();

        // Then - exits zero with the canonical "Project is valid" banner
        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('valid-project.txt');
    });

    test('--help prints the check command usage', async () => {
        // Given - any cwd; --help is resolved before the project walk
        const result = await spec('check help').project('empty').exec('check --help').run();

        // Then - cobra emits the usage block for the check subcommand
        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('help.txt');
    });

    test('freshly initialised project passes check cleanly', async () => {
        const result = await spec('init then check clean')
            .project('empty')
            .exec(['init --name init-check-test', 'check'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('Project is valid');
    });

    test('flags an agent that references a non-existent built-in tool', async () => {
        // Given - check-invalid-tool has tools: ["spwn:nonexistent"]
        const result = await spec('check invalid tool')
            .project('check-invalid-tool')
            .exec('check')
            .run();

        // Then - exits non-zero and lists the built-ins the user can pick from
        expect(result.exitCode).toBe(1);
        await result.stdout.toMatch('invalid-tool-ref.txt');
    });

    test('flags a remote-registry tool reference as unsupported', async () => {
        // Given - check-registry-tool has tools: ["@jterrazz/foo"]
        const result = await spec('check registry tool')
            .project('check-registry-tool')
            .exec('check')
            .run();

        // Then - exits non-zero with the "remote registries not yet supported" rule
        expect(result.exitCode).toBe(1);
        await result.stdout.toMatch('registry-not-supported.txt');
    });

    test('seed overlay flags the one-agent-one-world rule', async () => {
        // Given - single-agent base + a seed fragment that adds a second
        // World claiming the same neo agent. The framework's seed handler
        // Merges the YAML fragment into spwn.yaml inside the temp project.
        const result = await spec('one-agent-one-world')
            .project('single-agent')
            .seed('spwn.yaml/two-worlds-same-agent.yaml')
            .exec('check')
            .run();

        // Then - check fails with the rule violation, and the second
        // World is genuinely present in spwn.yaml on disk (proves the
        // Seed handler ran).
        expect(result.exitCode).toBe(1);
        expect(result.file('spwn.yaml').content).toContain('duplicate:');
        // `check` writes its rendered report to stdout (cmd.OutOrStdout).
        result.stdout.toContain('already deployed by world "duplicate"');
        result.stdout.toContain('spwn.yaml#worlds.neo.agents');
    });

    test('flags a skill missing the required YAML frontmatter', async () => {
        // Given - single-agent base + a naked skill dropped under
        // Spwn/agents/neo/skills/ via the framework's agent/ seed
        // Handler. The skill has no `--- name: ... ---` block.
        const result = await spec('skill frontmatter missing')
            .project('single-agent')
            .seed('agent/neo/skills/naked.md')
            .exec('check')
            .run();

        // Then - check exits non-zero, names the offending file, and
        // Hints at the header shape the user should add.
        expect(result.exitCode).toBe(1);
        expect(result.stdout.text).toContain('spwn/agents/neo/skills/naked.md');
        expect(result.stdout.text).toContain('missing YAML frontmatter');
        expect(result.stdout.text).toContain('name: <slug>');
        expect(result.stdout.text).toContain('description:');
    });

    test('emits a JSON report for a valid project', async () => {
        // Given - the frozen single-agent fixture
        const result = await spec('check json valid')
            .project('single-agent')
            .exec('check --json')
            .run();

        // Then - exits zero and emits the canonical JSON envelope
        expect(result.exitCode).toBe(0);
        await result.json.toMatch('valid.json');
    });

    test('emits a JSON report listing rule violations', async () => {
        // Given - check-invalid-tool references a nonexistent built-in
        const result = await spec('check json invalid tool')
            .project('check-invalid-tool')
            .exec('check --json')
            .run();

        // Then - exits non-zero and the issue list is structurally stable
        expect(result.exitCode).toBe(1);
        await result.json.toMatch('invalid-tool.json');
    });

    test('--deep passes on a valid project', async () => {
        // Given - the single-agent fixture has a real AGENTS.md so the
        // Compile pass finds nothing to complain about.
        const result = await spec('check deep valid')
            .project('single-agent')
            .exec('check --deep')
            .run();

        // Then - exits zero with the clean success banner. Deep adds
        // No issues on a healthy project.
        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('valid-project.txt');
    });

    test('--deep catches an empty AGENTS.md that shallow check misses', async () => {
        // Given - a seed wipes AGENTS.md to zero bytes. The manifest
        // Rule engine only checks that the file EXISTS, not that it
        // Has content, so a shallow check sees a valid project.
        const shallow = await spec('check deep shallow-pass')
            .project('single-agent')
            .seed('agent/neo/AGENTS.md')
            .exec('check')
            .run();
        expect(shallow.exitCode).toBe(0);

        // When - the same project is re-run with --deep
        const deep = await spec('check deep catches empty')
            .project('single-agent')
            .seed('agent/neo/AGENTS.md')
            .exec('check --deep')
            .run();

        // Then - deep exits non-zero and surfaces the empty prompt
        expect(deep.exitCode).toBe(1);
        const text = deep.stdout.text;
        expect(text).toContain('agent prompt is missing or empty');
        expect(text).toContain('spwn/agents/neo/AGENTS.md');
    });

    test('--deep --json tags compile issues with source=compile', async () => {
        const result = await spec('check deep json')
            .project('single-agent')
            .seed('agent/neo/AGENTS.md')
            .exec('check --deep --json')
            .run();

        expect(result.exitCode).toBe(1);
        const report = result.json.value as {
            issues: Array<{ level: string; message: string; source?: string }>;
            summary: { errors: number };
            valid: boolean;
        };
        expect(report.valid).toBe(false);
        const compileIssues = report.issues.filter((i) => i.source === 'compile');
        expect(compileIssues.length).toBeGreaterThan(0);
        expect(compileIssues[0].message).toContain('agent prompt');
    });

    test('errors when run outside a spwn project', async () => {
        // Given - the empty fixture has no spwn.yaml anywhere up the tree
        const result = await spec('check no project').project('empty').exec('check').run();

        // Then - exits non-zero and nudges the user at spwn init
        expect(result.exitCode).toBe(1);
        // `check` outside a project errors on stderr (project resolver).
        const stderr = result.stderr.text.toLowerCase();
        expect(stderr).toContain('spwn init');
        expect(stderr).toContain('spwn.yaml');
    });
});
