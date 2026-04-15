import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Zero-friction UX — the error paths and nudges the CLI emits when the
 * user hits a rough edge. These are loose assertions on substrings of
 * the error messages; full snapshots are reserved for happy paths.
 */

describe('zero-friction UX', () => {
    test('agent talk in a project whose agent has no running world nudges at spwn up', async () => {
        // Given - single-agent has neo declared but no live world
        const result = await spec('talk no world')
            .project('single-agent')
            .exec('agent talk neo hello')
            .run();

        // Then - the error walks the user to the spawn command
        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('talk-no-world.txt');
    });

    test('agent talk to a nonexistent agent suggests spwn agent create', async () => {
        // Given - empty project, no agents declared anywhere
        const result = await spec('talk missing agent')
            .project('empty')
            .exec('agent talk nonexistent hello')
            .run();

        // Then - exits non-zero with a "create one" hint
        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('talk-missing-agent-hint.txt');
    });

    test('architect stop when the daemon is not running exits cleanly', async () => {
        // Given - no architect container running (fresh temp cwd)
        const result = await spec('architect stop graceful')
            .project('empty')
            .exec('architect stop')
            .run();

        // Then - exits zero and the "not running" banner is rendered
        // To stderr. The point of the test is that stopping an
        // Already-stopped daemon is not an error.
        expect(result.exitCode).toBe(0);
        await result.stderr.toMatch('architect-stop-not-running.txt');
    });

    test('inspect nonexistent world gives ls hint', async () => {
        // Given - an empty project with no worlds
        const result = await spec('inspect missing hint')
            .project('empty')
            .exec('world inspect w-nonexistent-00000')
            .run();

        // Then - error points at spwn ls so the user can find their worlds
        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('inspect-missing-ls-hint.txt');
    });

    test('inspect missing world points at `spwn world list`, not the stale `spwn ls`', async () => {
        // Given - empty project, no worlds
        // When - inspecting a nonexistent world
        // Then - hint text references the correct list command
        const result = await spec('inspect missing correct hint')
            .project('empty')
            .exec('world inspect w-nonexistent-00000')
            .run();

        expect(result.exitCode).toBe(1);
        // Regression guard for QA finding #16: hint must point at
        // `spwn world list` (the real world-listing command). The
        // Old hint said `spwn ls`, which lists agents, not worlds.
        expect(result.stderr.text).toContain('spwn world list');
        expect(result.stderr.text).not.toMatch(/List worlds with: spwn ls$/m);
    });

    test('profile errors collapse $HOME to ~', async () => {
        // Given - an empty project pointed at an isolated SPWN_HOME.
        // When - we ask `spwn profile show` for a profile that does not exist.
        // Then - the error exits non-zero and the message does not leak an absolute
        //   /Users/... or /home/... host path (it should use ~ or the spwn-home dir).
        const result = await spec('profile show missing')
            .project('empty')
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec('profile show ghost')
            .run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).not.toMatch(/\/Users\/[^\s]+\.spwn/);
        expect(result.stderr.text).not.toMatch(/\/home\/[^\s]+\.spwn/);
        expect(result.stderr.text).toMatch(/ghost/);
    });

    test('architect talk --help lists usage and flags', async () => {
        // Given - --help is a pure cobra render, no side effects
        const result = await spec('architect talk help')
            .project('empty')
            .exec('architect talk --help')
            .run();

        // Then - exits zero with the talk usage block
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('spwn architect talk');
    });
});
