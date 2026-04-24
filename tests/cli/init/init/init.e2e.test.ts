import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn init` — project scaffolding. The legacy --global mode is gone;
 * these tests pin the modern project-mode behaviour:
 *
 *   - bare `init` in an empty dir writes the starter project tree
 *   - re-running in a populated dir errors unless --force is passed
 *   - `init spwn:matrix` installs a bundled example
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
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.file('spwn/agents/neo/AGENTS.md').exists).toBe(true);
        expect(result.file('.gitignore').exists).toBe(true);
        expect(result.file('.gitignore').content).toContain('.spwn');

        // One concrete example per local-ref scheme so users see how
        // Skill: and tool: are authored from their very first
        // `spwn init`. Regressions here mean the default project no
        // Longer demonstrates composition end-to-end.
        expect(result.file('spwn/skills/focus.md').exists).toBe(true);
        expect(result.file('spwn/skills/focus.md').content).toContain('name: focus');
        expect(result.file('spwn/tools/greet/tool.yaml').exists).toBe(true);
        expect(result.file('spwn/tools/greet/tool.yaml').content).toContain('name: greet');

        // Runtime hooks have their own top-level file — not a dep
        // Scheme. The scaffold ships one SessionStart example so the
        // Generic hooks.yaml → Claude/Codex translation has a live
        // Demo on day one.
        expect(result.file('spwn/hooks.yaml').exists).toBe(true);
        const hooksYaml = result.file('spwn/hooks.yaml').content;
        expect(hooksYaml).toContain('event: SessionStart');
        expect(hooksYaml).toContain('command:');

        // Default agent.yaml must reference the two dep-scheme
        // Examples so a fresh project shows the composition grammar
        // (spwn: + skill: + tool:) inline. The retired `hook:` scheme
        // Must NOT leak back into the scaffold.
        const agentYaml = result.file('spwn/agents/neo/agent.yaml').content;
        expect(agentYaml).toContain('skill:focus');
        expect(agentYaml).toContain('tool:greet');
        expect(agentYaml).not.toContain('hook:');
    });

    test('scaffolds the knowledge tree under spwn/, not at the project root', async () => {
        // Pin the 2026-04 relocation of the default knowledge path from
        // `./knowledge/` at the project root → `./spwn/knowledge/` so the
        // Whole spwn project is self-contained in one directory. A
        // Regression would put users back in the old layout and
        // Silently re-introduce a sibling dir at the repo root.
        const result = await spec('init knowledge default')
            .project('empty')
            .exec('init --name demo-project')
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);

        // The scaffolded knowledge dir lives under spwn/ and is kept
        // Alive in git via the .gitkeep sentinel.
        expect(result.file('spwn/knowledge').exists).toBe(true);
        expect(result.file('spwn/knowledge/.gitkeep').exists).toBe(true);

        // The legacy path (project-root ./knowledge/) must NOT be
        // Created. This is the anti-regression half — anyone
        // Restoring the old scaffold would trip this.
        expect(result.file('knowledge').exists).toBe(false);

        // And the manifest records the matching path so `spwn up`
        // Can resolve it. This is the user-facing contract.
        expect(result.file('spwn.yaml').content).toContain('knowledge: ./spwn/knowledge');
        expect(result.file('spwn.yaml').content).not.toMatch(/knowledge: \.\/knowledge$/m);
    });

    test('default scaffold omits runtime.backend (unpinned)', async () => {
        // Pin the 2026-04 decision to ship a blank runtime surface so
        // The resolver is free to pick based on auth state at spawn
        // Time. A regression would hard-pin `spwn:claude-code` again,
        // Silently locking new projects to claude-code even for users
        // Logged into codex.
        const result = await spec('init no-backend')
            .project('empty')
            .exec('init --name demo-project')
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml').content;

        // Walk every non-comment line; none should declare a runtime
        // Block or a backend key. Doing it this way (instead of a bare
        // Substring check) avoids catching the doc comment that
        // Explains the OPTIONAL runtime.backend surface.
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
        // User opts into a specific backend at scaffold time — the
        // Scaffolder canonicalises the short form to spwn:<name> and
        // Emits a runtime block. Captures the happy path end-to-end
        // Through the FS round-trip.
        const result = await spec('init backend codex')
            .project('empty')
            .exec('init --name codex-demo --backend codex')
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml').content;
        expect(agentYaml).toContain('runtime:');
        expect(agentYaml).toContain('backend: "spwn:codex"');
    });

    test('--backend accepts the spwn:<name> scheme form verbatim', async () => {
        // Mirrors the short-name test for authors who prefer the
        // Canonical catalog ref. Both forms must produce the same
        // Output line so tooling (grep, audits) can rely on a single
        // Shape.
        const result = await spec('init backend scheme')
            .project('empty')
            .exec('init --name scheme-demo --backend spwn:claude-code')
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);
        expect(result.file('spwn/agents/neo/agent.yaml').content).toContain(
            'backend: "spwn:claude-code"',
        );
    });

    test('--backend rejects unknown runtimes with a supported-list hint', async () => {
        // Guards against a typo silently producing an agent.yaml that
        // Fails later at spawn time with a less-specific error. The
        // Resolver's supported-list is the authoritative source of
        // Truth and must name each valid backend.
        const result = await spec('init backend unknown')
            .project('empty')
            .exec('init --name bad-demo --backend does-not-exist')
            .run();

        expect(result.exitCode).toBe(1);
        expect(result.file('spwn.yaml').exists).toBe(false);
        expect(result.stderr.text).toContain('unknown --backend');
        expect(result.stderr.text).toContain('claude-code');
        expect(result.stderr.text).toContain('codex');
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
        // Back into the example.
        expect(result.stdout.text).not.toContain('spwn/worlds/default.yaml');
        expect(result.file('spwn/worlds/default.yaml').exists).toBe(false);
    });

    test('init spwn:matrix installs the bundled example', async () => {
        // Given - an empty dir
        const result = await spec('init matrix').project('empty').exec('init spwn:matrix').run();

        // Then - the example's starter files land on disk
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/neo/agent.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        // The "Installed example spwn:matrix" banner lands on stderr.
        await result.stderr.toMatch('init-matrix-banner.txt');
        // And the stdout summary describes what was added.
        const out = result.stdout.text;
        expect(out).toContain('spwn.yaml');
        expect(out).toContain('Worlds added:');
        expect(out).toContain('matrix');
    });

    test('init matrix (bare) resolves to the catalog gallery entry', async () => {
        // Given - an empty dir
        // When - running with a bare catalog slug (no `spwn:` prefix)
        // Then - the bare name auto-resolves to spwn:matrix and the
        // Install proceeds exactly as the explicit form would. This is
        // The documentation-friendly shorthand: `spwn init matrix`.
        const result = await spec('init matrix bare').project('empty').exec('init matrix').run();

        expect(result.exitCode).toBe(0);
        expect(result.file('spwn.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        // Banner still prints the canonical spwn:matrix form even when
        // The user typed the bare slug — callers should never have to
        // Guess which scheme landed on disk.
        expect(result.stderr.text).toContain('Installed example spwn:matrix');
    });

    test('init <bare-miss> errors with the catalog hint', async () => {
        // Given - an empty dir
        // When - running init with a slug the catalog does not know
        // Then - exit 1, no spwn.yaml written, and the error names the
        // Available gallery entries so the user can correct their typo.
        const result = await spec('init bare miss').project('empty').exec('init nonesuch').run();

        expect(result.exitCode).toBe(1);
        expect(result.file('spwn.yaml').exists).toBe(false);
        expect(result.stderr.text).toContain('"nonesuch" is not in the catalog');
        // The hint must list real gallery entries — otherwise the user
        // Is left guessing which names are valid.
        expect(result.stderr.text).toMatch(/matrix|startup/);
    });

    test('scaffold passes spwn check --deep with no edits', async () => {
        // Sanity rail: whatever `spwn init` produces must pass the
        // Deep validator out of the box. If it doesn't, every new
        // User hits friction on their very first `check` command.
        // Deep mode runs the transpiler too, so this also catches
        // Any compile-time contract breakage in the scaffold.
        const result = await spec('init self-check')
            .project('empty')
            .exec(['init --name sanity-check', 'check --deep'])
            .run();

        expect(
            result.exitCode,
            `stdout:\n${result.stdout.text}\nstderr:\n${result.stderr.text}`,
        ).toBe(0);
        // On a clean project, the check command prints "Project is
        // Valid" and exits 0. If the scaffold ever regresses, the
        // Exit code flips non-zero and this test fails with the full
        // Report.
        expect(result.stdout.text).toContain('Project is valid');
    });

    test('init rejects tool-shaped catalog entries (not gallery-eligible)', async () => {
        // Given - an empty dir. `qmd` is a valid catalog tool entry
        // (--dep qmd works), but it has no `worlds:` section, so it's
        // Not installable via init.
        // When - running init qmd
        // Then - the resolver accepts the bare name up front (qmd is
        // NOT in ShippedSlugs), surfaces the gallery-entry hint, and
        // Leaves the directory empty.
        const result = await spec('init tool entry').project('empty').exec('init qmd').run();

        expect(result.exitCode).toBe(1);
        expect(result.file('spwn.yaml').exists).toBe(false);
        expect(result.stderr.text).toContain('not in the catalog');
    });
});
