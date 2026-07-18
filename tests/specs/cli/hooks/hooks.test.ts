import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Generic-hook translation. Each `spwn/hooks/<name>.yaml` the user authors
 * is translated into the runtime's native hook config at build time —
 * Claude Code's `.claude/settings.json` — but only for agents that
 * subscribed via `hook/<name>` in their agent.yaml#dependencies. The
 * hook-pilot neo agent declares both hooks. This spawns a real world,
 * reads the rendered settings from inside the container, and runs each
 * hook command via docker exec to prove the shell fragment is valid in
 * that environment (it does NOT drive a live Claude session — no network
 * / OAuth in CI). Every result binds with `await using` (rule B5).
 */

test('subscribed hooks land in .claude/settings.json and the commands run inside the world', async () => {
    // Given - the hook-pilot world brought online (real docker build path)
    await using result = await cli.fixture('$FIXTURES/hook-pilot/').exec('up');

    // Then - both subscribed hooks render into settings.json with the right shape and run inside the container
    expect(result.exitCode, `stdout:\n${result.stdout.text}\nstderr:\n${result.stderr.text}`).toBe(
        0,
    );
    const neo = result.container('neo');
    expect(neo.running).toBe(true);

    // The rendered settings file must exist at the agent's Claude config root
    expect(neo.file('/agents/neo/.claude/settings.json').exists).toBe(true);
    const settingsRaw = neo.file('/agents/neo/.claude/settings.json').content;
    const settings = JSON.parse(settingsRaw) as {
        hooks?: Record<
            string,
            Array<{ hooks: Array<{ command: string; type: string }>; matcher?: string }>
        >;
    };

    // SessionStart has no matcher on the source hook; the emitter writes the "*" fan-in matcher
    const sessionEntries = settings.hooks?.SessionStart ?? [];
    expect(sessionEntries.length).toBeGreaterThan(0);
    const sessionCmds = sessionEntries.flatMap((e) => e.hooks.map((h) => h.command));
    expect(sessionCmds.some((c) => c.includes('/tmp/hook-pilot-session.log'))).toBe(true);

    // PreToolUse source hook specifies matcher: Bash, which must land as the entry's matcher key
    const preToolUse = settings.hooks?.PreToolUse ?? [];
    const bashEntry = preToolUse.find((e) => e.matcher === 'Bash');
    expect(bashEntry, 'PreToolUse entry with matcher=Bash must be present').toBeDefined();
    expect(bashEntry?.hooks[0]?.command).toContain('/tmp/hook-pilot-bash.log');

    // Run each command directly inside the container to prove the shell fragment is valid there
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
