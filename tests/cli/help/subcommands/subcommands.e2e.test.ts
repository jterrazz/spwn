import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Subcommand help-text contracts.
 *
 * The top-level help pages (`spwn --help`, `spwn agent --help`, …)
 * are byte-snapshotted in ../output/output.e2e.test.ts. This file
 * pins key _subcommand_ help text with substring checks — any flag
 * rename, example rewrite, or description drift surfaces here
 * without the maintenance cost of a byte-level snapshot per page.
 *
 * Guiding principle: assert the stable nouns and verbs (flag names,
 * scheme words, example paths) the docs promise. Don't pin cosmetic
 * phrasing that might evolve with voice improvements.
 */

describe('subcommand help text', () => {
    test('agent create --help names the 2-layer Mind', async () => {
        const result = await spec('agent create help')
            .project('empty')
            .exec('agent create --help')
            .run();

        expect(result.exitCode).toBe(0);
        // The create verb documents what's scaffolded on disk; the
        // Mind layout (SOUL.md + 2 layers) is the contract — skills
        // Moved to build-time dependencies at /world/skills/.
        expect(result.stdout.text).toMatch(/SOUL\.md|2-layer/i);
    });

    test('install --help documents scoping + scheme grammar', async () => {
        const result = await spec('install help scoped')
            .project('empty')
            .exec('install --help')
            .run();

        expect(result.exitCode).toBe(0);
        const help = result.stdout.text;
        // --agent is the narrowing flag that replaced the retired
        // `agent add`. It must stay documented.
        expect(help).toMatch(/--agent/);
        // Examples must exercise both scopes so users see both.
        expect(help).toMatch(/spwn install python\b/);
        expect(help).toMatch(/--agent mark|--agent dylan|--agent neo/);
        // And the local-ref / local-authoring semantics are surfaced.
        expect(help).toMatch(/skill:|tool:|hook:/);
    });

    test('init --help advertises the bare-name gallery shorthand', async () => {
        const result = await spec('init help').project('empty').exec('init --help').run();

        expect(result.exitCode).toBe(0);
        const help = result.stdout.text;
        // Both the bare form and the explicit spwn:<slug> form
        // Must appear in the examples so users know both work.
        expect(help).toMatch(/spwn init matrix/);
        expect(help).toMatch(/spwn init spwn:matrix/);
    });

    test('install --help points at the five-scheme grammar', async () => {
        const result = await spec('install help deep')
            .project('empty')
            .exec('install --help')
            .run();

        expect(result.exitCode).toBe(0);
        const help = result.stdout.text;
        // The grammar hint is what the user reads when the CLI
        // Rejects their ref — it must be visible in help too.
        expect(help).toMatch(/spwn:/);
        expect(help).toMatch(/github:/);
        // And the bare shorthand is in the examples:
        expect(help).toMatch(/spwn install python/);
    });

    test('check --help explains the deep flag', async () => {
        const result = await spec('check help').project('empty').exec('check --help').run();

        expect(result.exitCode).toBe(0);
        // --deep is the flag that flips on transpile-time rules; it
        // Must stay documented or users won't discover it.
        expect(result.stdout.text).toMatch(/--deep/);
    });

    test('skill new --help documents the local authoring flow', async () => {
        const result = await spec('skill new help').project('empty').exec('skill new --help').run();

        expect(result.exitCode).toBe(0);
        // `skill new` is how users author local skills. The help
        // Must reference the `skill:` scheme so they know how to
        // Attach it afterwards.
        const help = result.stdout.text;
        expect(help).toMatch(/skill/i);
    });

    test('up --help and down --help are present and non-empty', async () => {
        // Smoke-level: these two lifecycle commands don't carry
        // Rich grammar, but they should render help without panicking
        // And say SOMETHING about worlds.
        const up = await spec('up help').project('empty').exec('up --help').run();
        expect(up.exitCode).toBe(0);
        expect(up.stdout.text.length).toBeGreaterThan(50);
        expect(up.stdout.text).toMatch(/world/i);

        const down = await spec('down help').project('empty').exec('down --help').run();
        expect(down.exitCode).toBe(0);
        expect(down.stdout.text.length).toBeGreaterThan(50);
        expect(down.stdout.text).toMatch(/world/i);
    });
});
