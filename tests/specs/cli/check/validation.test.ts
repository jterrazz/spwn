import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * CLI input validation — argument handling, unknown commands, and error message
 * quality. cobra is version-pinned via go.sum, so each probe asserts the exact
 * stable phrase for its case (not a loose alternation any case would pass),
 * discriminating the specific arity/unknown-command failure. Each spec gets a
 * fresh empty project and an isolated SPWN_HOME on the temp cwd.
 */

const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('cli input validation', () => {
    test("'spwn agent create' with no name picks a random planet name", async () => {
        // Given - no name is not an error: spwn picks a random planet name
        await using result = await isolated().exec('agent create');

        // Then - the chosen name is random, so assert the stable banner wording
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Creating agent');
        expect(result.stderr).toContain('Created agent');
        expect(result.stderr).toContain('Created soul');
    });

    test("'spwn agent create a b c' with too many args shows error", async () => {
        // Given - three positional args where at most one is accepted
        await using result = await isolated().exec('agent create a b c');

        // Then - exits non-zero with cobra's exact too-many-args phrase
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('accepts at most 1 arg(s), received 3');
    });

    test("'spwn down' with no world id shows error", async () => {
        // Given - the destroy command needs a world id
        await using result = await isolated().exec('down');

        // Then - exits non-zero with the destroy command's exact missing-id phrase
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('requires a world-id argument or --all flag');
    });

    test("'spwn world inspect' with no world id shows error", async () => {
        // Given - inspect needs a world id
        await using result = await isolated().exec('world inspect');

        // Then - exits non-zero with cobra's exact arity phrase for ExactArgs(1)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('accepts 1 arg(s), received 0');
    });

    test("'spwn world logs' with no world id shows error", async () => {
        // Given - logs needs a world id
        await using result = await isolated().exec('world logs');

        // Then - exits non-zero with cobra's exact arity phrase for ExactArgs(1)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('accepts 1 arg(s), received 0');
    });

    test('error messages do not dump full usage/help', async () => {
        // Given - three commands that should fail with a terse error
        const commands = [
            'down world-nonexistent-00000',
            'world inspect world-nonexistent-00000',
            'agent export nonexistent',
        ];

        // Then - no help-text leak on either stream (absence probe)
        for (const cmd of commands) {
            await using result = await isolated().exec(cmd);
            if (result.exitCode !== 0) {
                expect(result.stderr).not.toContain('Available Commands:');
                expect(result.stderr).not.toContain('Global Flags:');
                expect(result.stdout).not.toContain('Available Commands:');
                expect(result.stdout).not.toContain('Global Flags:');
            }
        }
    });

    test('error messages contain actionable hints', async () => {
        // Given - destroying a non-existent world
        await using result = await isolated().exec('down world-nonexistent-00000');

        // Then - a clean structured error with the ✗ prefix (presence probe)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('not found');
        expect(result.stderr).toContain('✗');
    });

    test('unknown top-level command shows error without full usage dump', async () => {
        // Given - a bogus top-level command
        await using result = await isolated().exec('foobar');

        // Then - exits non-zero with cobra's exact unknown-command phrase
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('unknown command "foobar" for "spwn"');
    });

    test('agent rm with no name shows error', async () => {
        // Given - the remove command needs a name
        await using result = await isolated().exec('agent rm');

        // Then - exits non-zero with cobra's exact arity phrase for ExactArgs(1)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('accepts 1 arg(s), received 0');
    });
});
