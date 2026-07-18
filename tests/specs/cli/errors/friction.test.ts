import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Zero-friction UX — the error paths and nudges the CLI emits when the user
 * hits a rough edge. The stable nudges are locked to stderr goldens; the
 * regression guard on the `spwn world list` hint stays a scalpel. Every result
 * binds with `await using` (rule B5).
 */

describe('zero-friction UX', () => {
    test('agent talk in a project whose agent has no running world nudges at spwn up', async () => {
        // Given - single-agent has neo declared but no live world
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .exec('agent talk neo hello');

        // Then - the error walks the user to the spawn command
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('talk-no-world.txt');
    });

    test('agent talk to a nonexistent agent suggests spwn agent create', async () => {
        // Given - empty project, no agents declared anywhere
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec('agent talk nonexistent hello');

        // Then - exits non-zero with a "create one" hint golden
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('talk-missing-agent-hint.txt');
    });

    test('architect stop when the daemon is not running exits cleanly', async () => {
        // Given - no architect container running (fresh temp cwd)
        await using result = await cli.fixture('$FIXTURES/empty/').exec('architect stop');

        // Then - exits zero; stopping an already-stopped daemon is not an error
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch('architect-stop-not-running.txt');
    });

    test('inspect nonexistent world gives ls hint', async () => {
        // Given - an empty project with no worlds
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec('world inspect world-nonexistent-00000');

        // Then - the error points at the world listing command
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('inspect-missing-ls-hint.txt');
    });

    test('inspect missing world points at `spwn world list`, not the stale `spwn ls`', async () => {
        // Given - empty project, no worlds; inspecting a nonexistent world
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec('world inspect world-nonexistent-00000');

        // Then - regression guard (QA #16): hint must name `spwn world list`, not the stale `spwn ls`
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toContain('spwn world list');
        expect(result.stderr.text).not.toMatch(/List worlds with: spwn ls$/m);
    });

    test('architect talk --help lists usage and flags', async () => {
        // Given - --help is a pure cobra render, no side effects
        await using result = await cli.fixture('$FIXTURES/empty/').exec('architect talk --help');

        // Then - exits zero and the talk usage line renders (cobra-formatted probe)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('spwn architect talk');
    });
});
