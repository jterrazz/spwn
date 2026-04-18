import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Snapshots of the grouped cobra help pages plus the --version banner.
 *
 * The help blocks are byte-stable — they live in Go strings and only
 * change when we intentionally reshuffle the CLI surface. Snapshot them.
 * The `auth` provider-table output touches the user's real keychain, so
 * it stays as a loose substring check against `result.stderr.text`.
 */

describe('CLI output', () => {
    test('root --help renders the grouped help page', async () => {
        // Given - any cwd; --help is resolved before the project walk
        const result = await spec('root help').project('empty').exec('--help').run();

        // Then - exits zero with the grouped help banner
        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('root-help.txt');
    });

    test('world --help lists the lifecycle subcommands', async () => {
        const result = await spec('world help').project('empty').exec('world --help').run();

        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('world-help.txt');
    });

    test('agent --help lists the agent subcommands', async () => {
        const result = await spec('agent help').project('empty').exec('agent --help').run();

        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('agent-help.txt');
    });

    test('architect --help lists the architect subcommands', async () => {
        const result = await spec('architect help').project('empty').exec('architect --help').run();

        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('architect-help.txt');
    });

    test('install --help describes the install flow', async () => {
        const result = await spec('install help').project('empty').exec('install --help').run();

        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('Add a catalog, GitHub, or local dependency');
    });

    test('upgrade --help describes the install flow', async () => {
        const result = await spec('upgrade help').project('empty').exec('upgrade --help').run();

        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('upgrade-help.txt');
    });

    test('auth --help lists the auth subcommands', async () => {
        const result = await spec('auth help').project('empty').exec('auth --help').run();

        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('auth-help.txt');
    });

    test('--version prints the spwn version line', async () => {
        // Given - the local build reports "dev" as its version
        const result = await spec('version').project('empty').exec('--version').run();

        // Then - output matches the `spwn version <v>` grammar
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toMatch(/spwn version /);
    });

    test('unknown command errors with a helpful cobra message', async () => {
        const result = await spec('unknown cmd').project('empty').exec('nonexistent').run();

        // Then - cobra writes "unknown command" to stderr and exits 1
        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('unknown-command.txt');
    });

    test('auth status table is rendered on stderr', async () => {
        // The provider table contents are keychain-dependent (reads the
        // Real anthropic keychain entry, the real ~/.codex/auth.json,
        // Etc.) so we match on the stable header row rather than a
        // Byte-for-byte snapshot.
        const result = await spec('auth status output')
            .project('empty')
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec('auth')
            .run();

        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toContain('PROVIDER');
        expect(stderr).toContain('STATUS');
        expect(stderr).toContain('SOURCE');
        expect(stderr).toContain('anthropic');
        expect(stderr).toContain('openai');
    });
});
