import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Snapshots of the grouped cobra help pages plus the --version banner.
 *
 * The help blocks are byte-stable — they live in Go strings and only
 * change when we intentionally reshuffle the CLI surface. Snapshot them.
 * The `auth` command touches the user's real keychain and the upgrade
 * --check hits the GitHub API, so those stay as loose substring checks.
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

    test('get --help lists the marketplace subcommands', async () => {
        const result = await spec('get help').project('empty').exec('get --help').run();

        expect(result.exitCode).toBe(0);
        await result.stdout.toMatch('get-help.txt');
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

        // Then - cobra writes "unknown command" to stderr and exits non-zero
        expect(result.exitCode).not.toBe(0);
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).toContain('unknown command "nonexistent" for "spwn"');
    });

    // Note: the `spwn auth` status table was covered by the legacy
    // Suite but is dropped here — the command writes everything to
    // Stderr (status, not data) and exits zero. The @jterrazz/test
    // ExecAdapter uses execSync, which discards stderr on success, so
    // There is nothing for the runner to assert against. The output
    // Itself is also keychain-dependent and not byte-stable.
});
