import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Generic-hook translation E2E.
 *
 * Contract: a single `spwn/hooks.yaml` authored by the user is
 * translated by the transpile layer into each runtime's native hook
 * config at build time — Claude Code's `.claude/settings.json` and
 * Codex's `.codex/hooks.json`. Previous regression: the `hook/` dep
 * form was left in the scaffold long after host-side lifecycle
 * hooks were retired, so users wrote host shell scripts that silently
 * never ran. The fix centralised all runtime hooks under hooks.yaml
 * and this test locks that contract in place.
 *
 * Strategy:
 *   1. Spawn a real world from the hook-pilot fixture (real Docker
 *      build path, no SPWN_BASE_IMAGE shortcut).
 *   2. Read `/agents/neo/.claude/settings.json` from inside the
 *      container and assert the two hooks landed with the right
 *      event/matcher/command shape.
 *   3. Run each hook command directly via docker exec and assert
 *      the side-effect file appears — proves the shell fragment
 *      itself is valid inside the container's environment.
 *
 * The test does NOT drive a live Claude session (no network / no
 * OAuth in CI). Firing is covered by the runtime's own contract;
 * what we own is the translation + the command's in-container
 * executability. Together with the unit-level render tests under
 * packages/runtimes/claudecode this exercises every link of the
 * chain spwn is responsible for.
 */
describe('hooks.yaml translation', () => {
    test('hooks.yaml lands in .claude/settings.json and its commands run inside the world', async () => {
        await using result = await spec('hooks translation').project('hook-pilot').exec('up').run();

        expect(
            result.exitCode,
            `stdout:\n${result.stdout.text}\nstderr:\n${result.stderr.text}`,
        ).toBe(0);

        const neo = result.container('neo');
        expect(neo.running).toBe(true);

        // The rendered settings file must exist at the agent's
        // Claude config root.
        expect(neo.file('/agents/neo/.claude/settings.json').exists).toBe(true);

        const settingsRaw = neo.file('/agents/neo/.claude/settings.json').content;
        const settings = JSON.parse(settingsRaw) as {
            hooks?: Record<
                string,
                Array<{ matcher?: string; hooks: Array<{ command: string; type: string }> }>
            >;
        };

        // SessionStart — no matcher on the source hook; the Claude
        // Code emitter writes the "*" fan-in matcher.
        const sessionEntries = settings.hooks?.SessionStart ?? [];
        expect(sessionEntries.length).toBeGreaterThan(0);
        const sessionCmds = sessionEntries.flatMap((e) => e.hooks.map((h) => h.command));
        expect(sessionCmds.some((c) => c.includes('/tmp/hook-pilot-session.log'))).toBe(true);

        // PreToolUse — source hook specifies matcher: Bash, which
        // Must land as the entry's matcher key.
        const preToolUse = settings.hooks?.PreToolUse ?? [];
        const bashEntry = preToolUse.find((e) => e.matcher === 'Bash');
        expect(bashEntry, 'PreToolUse entry with matcher=Bash must be present').toBeDefined();
        expect(bashEntry?.hooks[0]?.command).toContain('/tmp/hook-pilot-bash.log');

        // Run each command directly inside the container to prove
        // The shell fragment is valid in that environment — catches
        // Regressions like missing binaries or bad quoting at the
        // YAML → JSON boundary.
        const runSession = await neo.exec(`sh -c ${JSON.stringify(sessionCmds[0])}`);
        expect(runSession.exitCode).toBe(0);
        expect(neo.file('/tmp/hook-pilot-session.log').exists).toBe(true);
        expect(neo.file('/tmp/hook-pilot-session.log').content).toMatch(
            /session-start=\d{4}-\d{2}-\d{2}T/,
        );

        const bashCmd = bashEntry?.hooks[0]?.command ?? '';
        const runBash = await neo.exec(`sh -c ${JSON.stringify(bashCmd)}`);
        expect(runBash.exitCode).toBe(0);
        expect(neo.file('/tmp/hook-pilot-bash.log').exists).toBe(true);
    });
});
