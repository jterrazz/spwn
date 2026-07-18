import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Error-handling — one test per error shape, each paired with a locked-in
 * stderr golden under `expected/<name>.txt`. spwn writes the "→ Doing X..." /
 * "✗ Failed ..." banners to stderr, captured on both the success and failure
 * paths. The runner is docker-aware, so every result binds with `await using`
 * even though these spawn no containers (rule B5).
 */

const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('error handling', () => {
    test('destroy non-existent world', async () => {
        // Given - an isolated empty project
        await using result = await isolated().exec('down world-nonexistent-00000');

        // Then - exits non-zero with the canonical destroy-failed banner
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('destroy-missing.txt');
    });

    test('inspect non-existent world', async () => {
        // Given - an isolated empty project
        await using result = await isolated().exec('world inspect world-nonexistent-00000');

        // Then - exits non-zero with the missing-world error golden
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('inspect-missing.txt');
    });

    test('agent --ephemeral without --world flag', async () => {
        // Given - an ephemeral agent request with no world to attach to
        await using result = await isolated().exec('agent --ephemeral lint-code');

        // Then - exits non-zero
        expect(result.exitCode).toBe(1);
    });

    test('agent dream non-existent agent skips gracefully', async () => {
        // Given - an isolated empty project with no agents
        await using result = await isolated().exec('agent dream nonexistent');

        // Then - exits zero with the "skipped, no journal" golden
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch('dream-missing.txt');
    });

    test('agent export non-existent agent', async () => {
        // Given - an isolated empty project with no agents
        await using result = await isolated().exec('agent export nonexistent');

        // Then - exits non-zero with the export-failed golden
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('export-missing.txt');
    });

    test('logs for non-existent world', async () => {
        // Given - `spwn world logs` filters by id; a missing world yields no events
        await using result = await isolated().exec('world logs world-nonexistent-00000');

        // Then - the command does not crash (absence probe: no panic/goroutine dump)
        expect(result.stderr).not.toContain('panic:');
        expect(result.stderr).not.toContain('goroutine');
    });

    test('agent talk to non-existent agent', async () => {
        // Given - an isolated empty project with no agents
        await using result = await isolated().exec('agent talk nonexistent "hello"');

        // Then - exits non-zero with the talk-failed golden
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('talk-missing.txt');
    });

    test('delete non-existent agent shows error', async () => {
        // Given - an isolated empty project with no agents
        await using result = await isolated().exec('agent rm ghost');

        // Then - exits non-zero with the delete-failed golden
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('delete-missing.txt');
    });

    test('no usage dump on errors', async () => {
        // Given - a command that fails against a missing world
        await using result = await isolated().exec('down world-nonexistent-00000');

        // Then - errors must not leak help text into either stream (absence probe)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).not.toContain('Available Commands:');
        expect(result.stderr).not.toContain('Global Flags:');
        expect(result.stderr).not.toContain('Use "spwn');
        expect(result.stdout).not.toContain('Available Commands:');
        expect(result.stdout).not.toContain('Global Flags:');
        expect(result.stdout).not.toContain('Use "spwn');
    });

    test('error messages follow the structured ✗ convention', async () => {
        // Given - the same missing-world destroy as above, reusing its golden to anchor wording
        await using result = await isolated().exec('down world-nonexistent-00000');

        // Then - the structured destroy-failed banner matches
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('destroy-missing.txt');
    });
});
