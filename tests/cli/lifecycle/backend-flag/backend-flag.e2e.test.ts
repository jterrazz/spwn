import { describe, expect, test } from 'vitest';

import { spec } from '../../../_setup/cli.specification.js';

/**
 * Runtime backend override flag (--backend) on every spawn entry point.
 *
 * Before this flag shipped, users logged into multiple providers had
 * No way to override runtime resolution at the CLI level — they had
 * To pin in agent.yaml or spwn.yaml. The resolver's disambiguation
 * Hint pointed at `--runtime` which only existed on `spwn build`, so
 * Copy-pasting the hint produced "unknown flag" errors.
 *
 * These tests pin that the flag is registered on every command that
 * Advertises it, by checking either `--help` output or that a command
 * Fails with its expected error (not cobra's "unknown flag") when the
 * Flag is passed.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('CLI - --backend flag', () => {
    test("'spwn up --help' advertises the --backend flag", async () => {
        const result = await isolated('up help backend').exec('up --help').run();

        expect(result.exitCode).toBe(0);
        const out = result.stdout.text;
        expect(out).toContain('--backend');
        // The help string names both canonical runtimes so users
        // Know valid values without leaving the help page.
        expect(out).toContain('claude-code');
        expect(out).toContain('codex');
    });

    test("'spwn world up --help' advertises the --backend flag", async () => {
        // `registerSpawnFlags` is shared between `spwn up` (top-level
        // Alias) and `spwn world up` (grammar form). Both must expose
        // The same surface.
        const result = await isolated('world up help backend').exec('world up --help').run();

        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('--backend');
    });

    test("'spwn agent <nosuch> --backend X' parses the flag (fails on agent, not on flag)", async () => {
        // `spwn agent <name>` is the ergonomic shortcut. Before this
        // Fix, passing --backend here bombed with cobra's "unknown
        // Flag" error because the flag lived only on the world
        // Subcommands. Running against a non-existent agent proves
        // The flag parses — we reach the ValidateMind step, which is
        // Past cobra's flag-validation phase.
        const result = await isolated('agent bare backend parses')
            .exec('agent nonexistent --backend codex')
            .run();

        expect(result.exitCode).toBe(1);
        const err = result.stderr.text;
        expect(err).not.toContain('unknown flag');
        expect(err).toContain('agent "nonexistent" not found');
    });
});
