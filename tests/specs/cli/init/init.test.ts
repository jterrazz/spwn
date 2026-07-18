import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn init` — project scaffolding (project mode; the legacy --global mode is
 * gone). The deterministic banners are pinned as goldens; the on-disk contract
 * is asserted through `.file()`. We pass --name so the stdout/stderr banners
 * stay stable across runs. Every result binds with `await using` (rule B5); no
 * docker.
 */

describe('spwn init', () => {
    test('scaffolds the starter project in an empty directory', async () => {
        // Given - the empty fixture scaffolded with a fixed name
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init --name demo-project');

        // Then - the banners match their goldens and the key files land on disk
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch('init-banner.txt');
        expect(result.stdout).toMatch('init-output.txt');

        expect(result.file('spwn.yaml').exists).toBe(true);
        expect(result.file('spwn.yaml').content).toContain('name: demo-project');
        expect(result.file('spwn/agents/neo/agent.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.file('spwn/agents/neo/AGENTS.md').exists).toBe(true);
        expect(result.file('.gitignore').exists).toBe(true);
        expect(result.file('.gitignore').content).toContain('.spwn');

        // And - one concrete example per local-ref scheme ships in the scaffold
        expect(result.file('spwn/skills/focus.md').exists).toBe(true);
        expect(result.file('spwn/skills/focus.md').content).toContain('name: focus');
        expect(result.file('spwn/tools/greet/tool.yaml').exists).toBe(true);
        expect(result.file('spwn/tools/greet/tool.yaml').content).toContain('name: greet');
        expect(result.file('spwn/hooks/session-banner.yaml').exists).toBe(true);
        const hookYaml = result.file('spwn/hooks/session-banner.yaml').content;
        expect(hookYaml).toContain('event: SessionStart');
        expect(hookYaml).toContain('command:');
        expect(result.file('spwn/hooks.yaml').exists).toBe(false);
        expect(result.file('spwn/commands/refactor.md').exists).toBe(true);
        const cmdMD = result.file('spwn/commands/refactor.md').content;
        expect(cmdMD).toContain('Refactor');
        expect(cmdMD).toContain('description:');

        // And - the default agent.yaml references all local-ref examples, no colon-form leaks
        const agentYaml = result.file('spwn/agents/neo/agent.yaml').content;
        expect(agentYaml).toContain('skill/focus');
        expect(agentYaml).toContain('tool/greet');
        expect(agentYaml).toContain('hook/session-banner');
        expect(agentYaml).toContain('command/refactor');
        expect(agentYaml).not.toMatch(/\bskill:[a-z]/);
        expect(agentYaml).not.toMatch(/\btool:[a-z]/);
        expect(agentYaml).not.toMatch(/\bhook:[a-z]/);
    });

    test('scaffolds the knowledge tree under spwn/, not at the project root', async () => {
        // Given - a fresh scaffold pinning the 2026-04 knowledge relocation to spwn/knowledge
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init --name demo-project');

        // Then - the knowledge dir lives under spwn/ with a .gitkeep sentinel, never at root
        expect(result.exitCode, result.stderr.text).toBe(0);
        expect(result.file('spwn/knowledge').exists).toBe(true);
        expect(result.file('spwn/knowledge/.gitkeep').exists).toBe(true);
        expect(result.file('knowledge').exists).toBe(false);
        expect(result.file('spwn.yaml').content).toContain('knowledge: ./spwn/knowledge');
        expect(result.file('spwn.yaml').content).not.toMatch(/knowledge: \.\/knowledge$/m);
    });

    test('default scaffold omits runtime.backend (unpinned)', async () => {
        // Given - a fresh scaffold; the resolver should pick the backend at spawn time
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init --name demo-project');

        // Then - no non-comment line declares a runtime block or backend key
        expect(result.exitCode, result.stderr.text).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml').content;
        for (const raw of agentYaml.split('\n')) {
            const line = raw.trimStart();
            if (line.startsWith('#')) {
                continue;
            }
            expect(line).not.toMatch(/^runtime:\s*$/);
            expect(line).not.toMatch(/^backend:/);
        }
    });

    test('--backend writes runtime.backend into the scaffolded agent', async () => {
        // Given - opting into a backend at scaffold time
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec('init --name codex-demo --backend codex');

        // Then - the short form is canonicalised to spwn:codex in a runtime block
        expect(result.exitCode, result.stderr.text).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml').content;
        expect(agentYaml).toContain('runtime:');
        expect(agentYaml).toContain('backend: "spwn:codex"');
    });

    test('--backend accepts the spwn:<name> scheme form verbatim', async () => {
        // Given - the canonical catalog ref passed to --backend
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec('init --name scheme-demo --backend spwn:claude-code');

        // Then - the same output line is produced as the short form
        expect(result.exitCode, result.stderr.text).toBe(0);
        expect(result.file('spwn/agents/neo/agent.yaml').content).toContain(
            'backend: "spwn:claude-code"',
        );
    });

    test('--backend rejects unknown runtimes with a supported-list hint', async () => {
        // Given - a typo'd backend
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec('init --name bad-demo --backend does-not-exist');

        // Then - exits non-zero, writes no manifest, and names the valid backends
        expect(result.exitCode).toBe(1);
        expect(result.file('spwn.yaml').exists).toBe(false);
        expect(result.stderr).toContain('unknown --backend');
        expect(result.stderr).toContain('claude-code');
        expect(result.stderr).toContain('codex');
    });

    test('errors when spwn.yaml already exists', async () => {
        // Given - the single-agent fixture already has spwn.yaml
        await using result = await cli.fixture('$FIXTURES/single-agent/').exec('init');

        // Then - exits non-zero pointing at --force (golden)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('init-already-exists.txt');
    });

    test('--force overwrites an existing spwn.yaml', async () => {
        // Given - the single-agent fixture with a populated tree
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .exec('init --force --name forced-project');

        // Then - exits zero and rewrites spwn.yaml with the passed name
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn.yaml').content).toContain('name: forced-project');
    });

    test('init rejects --name values that fail the manifest regex', async () => {
        // Given - a --name with spaces
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init --name "Has Spaces"');

        // Then - exits non-zero, writes no manifest, and complains about the name (error probe)
        expect(result.exitCode).toBe(1);
        expect(result.file('spwn.yaml').exists).toBe(false);
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/invalid --name/i);
    });

    test('init banner only references files that actually exist', async () => {
        // Given - an empty dir scaffolded via init
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init --name demo-project');

        // Then - the removed spwn/worlds/default.yaml path never sneaks back (absence probe)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).not.toContain('spwn/worlds/default.yaml');
        expect(result.file('spwn/worlds/default.yaml').exists).toBe(false);
    });

    test('init spwn:matrix installs the bundled example', async () => {
        // Given - an empty dir with the matrix example installed
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init spwn:matrix');

        // Then - the example lands on disk, the banner golden matches, the summary describes it
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/neo/agent.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.stderr).toMatch('init-matrix-banner.txt');
        expect(result.stdout).toContain('spwn.yaml');
        expect(result.stdout).toContain('Worlds added:');
        expect(result.stdout).toContain('matrix');
    });

    test('init matrix (bare) resolves to the catalog gallery entry', async () => {
        // Given - a bare catalog slug with no spwn: prefix
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init matrix');

        // Then - the bare name auto-resolves to spwn:matrix and installs (banner names the canonical form)
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.stderr).toContain('Installed example spwn:matrix');
    });

    test('init <bare-miss> errors with the catalog hint', async () => {
        // Given - a slug the catalog does not know
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init nonesuch');

        // Then - exits non-zero, writes no manifest, and lists real gallery entries (error probe)
        expect(result.exitCode).toBe(1);
        expect(result.file('spwn.yaml').exists).toBe(false);
        expect(result.stderr).toContain('"nonesuch" is not in the catalog');
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/matrix|startup/);
    });

    test('scaffold passes spwn check --deep with no edits', async () => {
        // Given - a fresh scaffold checked deep in one chain
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init --name sanity-check', 'check --deep']);

        // Then - the deep validator passes out of the box
        expect(
            result.exitCode,
            `stdout:\n${result.stdout.text}\nstderr:\n${result.stderr.text}`,
        ).toBe(0);
        expect(result.stdout).toContain('Project is valid');
    });

    test('init rejects tool-shaped catalog entries (not gallery-eligible)', async () => {
        // Given - qmd is a valid catalog tool but has no worlds: section, so it's not init-able
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init qmd');

        // Then - exits non-zero, writes no manifest, and surfaces the gallery-entry hint (error probe)
        expect(result.exitCode).toBe(1);
        expect(result.file('spwn.yaml').exists).toBe(false);
        expect(result.stderr).toContain('not in the catalog');
    });
});
