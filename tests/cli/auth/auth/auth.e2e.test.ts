import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Spwn auth — credential / provider status flows.
 *
 * Every test runs with an isolated SPWN_HOME so we never touch the
 * user's real keychain or credentials file. No real network calls
 * are made; the only subcommands we exercise are status / check /
 * help / logout, which are safe on a fresh home.
 *
 * `spwn auth` renders its provider table to stderr (Unix convention
 * for status output). The table content is keychain/OS-dependent, so
 * we assert on intent-level substrings against `result.stderr.text`
 * rather than snapshotting. The `auth logout` messages are stable
 * and get full stderr snapshots.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('CLI - auth command', () => {
    test("'spwn auth' prints the provider status table", async () => {
        const result = await isolated('auth status').exec('auth').run();

        // The provider table is keychain-dependent (the anthropic row
        // Reads Claude Code's real keychain entry on the dev box) so
        // We assert on the stable header row rather than a snapshot.
        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toContain('PROVIDER');
        // The status-table column header is `STATE` (active | known |
        // None); it was briefly called `STATUS` in an earlier revision.
        expect(stderr).toContain('STATE');
        expect(stderr).toContain('anthropic');
        expect(stderr).toContain('openai');
    });

    test("'spwn auth check' validates credentials against each provider", async () => {
        const result = await isolated('auth check').exec('auth check').run();

        // Validation output is keychain/network-dependent. Intent: the
        // Validate banner fires and the provider table is rendered.
        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toContain('Validating credentials');
        expect(stderr).toContain('PROVIDER');
        expect(stderr).toContain('anthropic');
    });

    test("'spwn auth --help' shows subcommands", async () => {
        const result = await isolated('auth help').exec('auth --help').run();

        expect(result.exitCode).toBe(0);
        const out = result.stdout.text;
        expect(out).toContain('auth');
        const hasSubcommands =
            out.includes('check') ||
            out.includes('token') ||
            out.includes('login') ||
            out.includes('logout') ||
            out.includes('Commands') ||
            out.includes('COMMANDS') ||
            out.includes('Usage');
        expect(hasSubcommands).toBe(true);
    });

    test("'spwn auth token --help' shows usage", async () => {
        const result = await isolated('auth token help').exec('auth token --help').run();

        expect(result.exitCode).toBe(0);
        const out = result.stdout.text;
        expect(out).toContain('token');
        const hasUsage =
            out.includes('Usage') ||
            out.includes('usage') ||
            out.includes('USAGE') ||
            out.includes('Options') ||
            out.includes('--help');
        expect(hasUsage).toBe(true);
    });

    /*
     * SKIPPED: `spwn auth login` enters an interactive prompt reading
     * from stdin; the ExecAdapter has no way to pipe empty input / EOF
     * reliably without hanging. See tests/setup/spwn.specification.ts.
     */
    test.skip("'spwn auth login' handles non-interactive gracefully", () => {});

    /*
     * SKIPPED: `spwn auth logout <provider>` now requires a provider
     * argument, and the keychain-backed anthropic credential is read
     * and mutated from the OS keychain — SPWN_HOME isolation does
     * NOT stub that out. Running this test against the real keychain
     * deletes the user's Claude Code OAuth token as collateral
     * damage. Re-enable once the auth layer grows a keychain-stub
     * mode (e.g. SPWN_KEYCHAIN_BACKEND=memory) that tests can opt
     * into. Until then, logout behaviour is covered manually.
     */
    test.skip("'spwn auth logout <provider>' on a fresh home emits the no-op banner", () => {});

    test.skip("'spwn auth logout <provider>' is idempotent", () => {});
});
