import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Zero-friction UX — the error paths and nudges the CLI emits when the
 * user hits a rough edge. These are loose assertions on substrings of
 * the error messages; full snapshots are reserved for happy paths.
 *
 * Error messages are written to stderr; the @jterrazz/test ExecAdapter
 * only captures stderr when the command exits non-zero, so tests that
 * need to read a successful command's stderr fall back to asserting on
 * exitCode only.
 */

describe('zero-friction UX', () => {
    test('agent talk in a project whose agent has no running world nudges at spwn up', async () => {
        // Given - single-agent has neo declared but no live world
        const result = await spec('talk no world')
            .project('single-agent')
            .exec('agent talk neo hello')
            .run();

        // Then - the error walks the user to the spawn command
        expect(result.exitCode).not.toBe(0);
        const out = result.stdout.text + result.stderr.text;
        expect(out).toContain('not in any active world');
        expect(out).toContain('spwn up --agent neo');
    });

    test('agent talk to a nonexistent agent suggests spwn agent create', async () => {
        // Given - empty project, no agents declared anywhere
        const result = await spec('talk missing agent')
            .project('empty')
            .exec('agent talk nonexistent hello')
            .run();

        // Then - exits non-zero with a "create one" hint
        expect(result.exitCode).not.toBe(0);
        const out = result.stdout.text + result.stderr.text;
        expect(out).toContain('agent "nonexistent" not found');
        expect(out).toContain('spwn agent create nonexistent');
    });

    test('architect stop when the daemon is not running exits cleanly', async () => {
        // Given - no architect container running (fresh temp cwd)
        const result = await spec('architect stop graceful')
            .project('empty')
            .exec('architect stop')
            .run();

        // Then - exits zero. The "not running" banner is on stderr and
        // The ExecAdapter discards stderr on success, so we only
        // Assert the exit code here — the point of the test is that
        // Stopping an already-stopped daemon is not an error.
        expect(result.exitCode).toBe(0);
    });

    test('inspect nonexistent world gives ls hint', async () => {
        // Given - an empty project with no worlds
        const result = await spec('inspect missing hint')
            .project('empty')
            .exec('world inspect w-nonexistent-00000')
            .run();

        // Then - error points at spwn ls so the user can find their worlds
        expect(result.exitCode).not.toBe(0);
        const out = result.stdout.text + result.stderr.text;
        expect(out).toContain('w-nonexistent-00000');
        expect(out).toContain('spwn ls');
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
