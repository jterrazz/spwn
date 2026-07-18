import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Runtime backend override flag (--backend) on the spawn entry points. Both the
 * top-level `spwn up` alias and the `spwn world up` grammar form share
 * `registerSpawnFlags`, so their deterministic --help surfaces are pinned as
 * full goldens. The bare-agent path proves the flag parses (fails on the agent,
 * not on the flag). The runner is docker-aware, so every result binds with
 * `await using` even though nothing spawns a container (rule B5).
 */

const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('cli --backend flag', () => {
    test('spwn up --help advertises the --backend flag', async () => {
        // Given - the top-level up alias help
        await using result = await isolated().exec('up --help');

        // Then - cobra prints the full up usage block, which documents --backend
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('up-help.txt');
    });

    test('spwn world up --help advertises the --backend flag', async () => {
        // Given - the grammar-form world up help
        await using result = await isolated().exec('world up --help');

        // Then - cobra prints the full world-up usage block sharing the same flag surface
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('world-up-help.txt');
    });

    test('spwn agent <nosuch> --backend X parses the flag, failing on the agent', async () => {
        // Given - the ergonomic agent shortcut passed --backend against a missing agent
        await using result = await isolated().exec('agent nonexistent --backend codex');

        // Then - reaches ValidateMind (past flag parsing): no cobra flag error, an agent error instead (scalpel: error wording)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).not.toContain('unknown flag');
        expect(result.stderr).toContain('agent "nonexistent" not found');
    });
});
