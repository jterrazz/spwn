import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Goldens of the grouped cobra help pages plus the --version banner. The help
 * blocks are byte-stable — they live in Go strings and only change when we
 * intentionally reshuffle the CLI surface, so they are full goldens. The
 * `auth` dashboard touches the user's real keychain, so it stays a substring
 * probe. Every result binds with `await using` (rule B5).
 */

describe('cli output', () => {
    test('root --help renders the grouped help page', async () => {
        // Given - any cwd; --help is resolved before the project walk
        await using result = await cli.fixture('$FIXTURES/empty/').exec('--help');

        // Then - exits zero with the grouped help golden
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('root-help.txt');
    });

    test('world --help lists the lifecycle subcommands', async () => {
        // Given - the world command group
        await using result = await cli.fixture('$FIXTURES/empty/').exec('world --help');

        // Then - the world help golden matches
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('world-help.txt');
    });

    test('agent --help lists the agent subcommands', async () => {
        // Given - the agent command group
        await using result = await cli.fixture('$FIXTURES/empty/').exec('agent --help');

        // Then - the agent help golden matches
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('agent-help.txt');
    });

    test('architect --help lists the architect subcommands', async () => {
        // Given - the architect command group
        await using result = await cli.fixture('$FIXTURES/empty/').exec('architect --help');

        // Then - the architect help golden matches
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('architect-help.txt');
    });

    test('install --help describes the install flow', async () => {
        // Given - the install command help
        await using result = await cli.fixture('$FIXTURES/empty/').exec('install --help');

        // Then - the summary line renders (cobra-formatted probe; no dedicated golden)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('Add a catalog, GitHub, or local dependency');
    });

    test('upgrade --help describes the install flow', async () => {
        // Given - the upgrade command help
        await using result = await cli.fixture('$FIXTURES/empty/').exec('upgrade --help');

        // Then - the upgrade help golden matches
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('upgrade-help.txt');
    });

    test('auth --help lists the auth subcommands', async () => {
        // Given - the auth command group
        await using result = await cli.fixture('$FIXTURES/empty/').exec('auth --help');

        // Then - the auth help golden matches
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('auth-help.txt');
    });

    test('--version prints the spwn version line', async () => {
        // Given - the local build reports its version
        await using result = await cli.fixture('$FIXTURES/empty/').exec('--version');

        // Then - output matches the `spwn version <v>` grammar (scalpel: version is dynamic)
        expect(result.exitCode).toBe(0);
        const stdout = result.stdout.text;
        expect(stdout).toMatch(/spwn version /);
    });

    test('unknown command errors with a helpful cobra message', async () => {
        // Given - a bogus top-level command
        await using result = await cli.fixture('$FIXTURES/empty/').exec('nonexistent');

        // Then - cobra writes "unknown command" to stderr and exits 1 (golden)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('unknown-command.txt');
    });

    test('auth dashboard renders on stderr', async () => {
        // Given - an isolated home so the dashboard never hits real providers
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec('auth');

        // Then - the stable scaffolding is present (scalpel: row content is keychain-dependent)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Credentials');
        expect(result.stderr).toContain('Anthropic');
        expect(result.stderr).toContain('OpenAI');
        expect(result.stderr).toContain('Default:');
    });
});
