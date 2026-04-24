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

    test("'spwn auth default' on a fresh home reports 'not set'", async () => {
        // Durable soft-preference for multi-provider ambiguity. Fresh
        // Home → no default → the hint guides the user to the setter.
        const result = await isolated('auth default unset').exec('auth default').run();

        expect(result.exitCode).toBe(0);
        const err = result.stderr.text;
        expect(err).toContain('default provider:');
        expect(err).toContain('not set');
        expect(err).toContain('spwn auth default <provider>');
    });

    test("'spwn auth default <provider>' sets the preference", async () => {
        // Set + verify the banner on its own. Persistence round-trip
        // Is covered by the follow-up test below so each assertion
        // Pins one concern; the test framework exposes only the final
        // Command's stderr, so we can't sensibly test both in one run.
        const result = await isolated('auth default set').exec('auth default anthropic').run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);
        expect(result.stderr.text).toContain('default provider set to anthropic');

        // File exists and carries the field — a durable-format check
        // So future yaml shape changes don't silently break the
        // Resolver's read path.
        const authYaml = result.file('spwn-home/auth.yaml').content;
        expect(authYaml).toContain('default_provider: anthropic');
    });

    test("'spwn auth default' reads back what was previously set", async () => {
        // Round-trip through auth.yaml on SPWN_HOME. Both commands
        // Share the workdir (and therefore SPWN_HOME) so the second
        // Call sees the first call's write. We only see the final
        // Command's stderr — that's fine, the final read proves the
        // Value was persisted.
        const result = await isolated('auth default round-trip')
            .exec(['auth default anthropic', 'auth default'])
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);
        const err = result.stderr.text;
        expect(err).toContain('default provider:');
        expect(err).toContain('anthropic');
        // Info's label column is right-padded so the hint line reads
        // "Clear with:   spwn auth default --clear" — regex collapses
        // The variable whitespace.
        expect(err).toMatch(/Clear with:\s+spwn auth default --clear/);
    });

    test("'spwn auth default --clear' unsets the preference", async () => {
        // Verify --clear directly by running it after a set and
        // Inspecting the auth.yaml file — the final command is the
        // Clear itself so its stderr carries the success banner. The
        // File assertion is the durable proof: the field must not
        // Remain for the resolver to read it.
        const result = await isolated('auth default clear')
            .exec(['auth default openai', 'auth default --clear'])
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);
        expect(result.stderr.text).toContain('default provider cleared');

        // File was rewritten — if the field is present it's empty.
        const authYaml = result.file('spwn-home/auth.yaml').content;
        expect(authYaml).not.toMatch(/default_provider:\s*openai/);
    });

    test("'spwn auth default' refuses a disabled provider", async () => {
        // Refusing makes the auth layer internally consistent: a
        // Disabled provider is invisible to the resolver anyway, so
        // Picking it as default would silently do nothing. Loud error
        // Beats subtle no-op.
        const result = await isolated('auth default disabled')
            .exec(['auth disable openai', 'auth default openai'])
            .run();

        expect(result.exitCode).toBe(1);
        const err = result.stderr.text;
        expect(err).toContain('is currently disabled');
        expect(err).toContain('spwn auth enable openai');
    });

    test("'spwn auth' status table shows the default provider when set", async () => {
        // Dashboard contract: once a default exists, it must be
        // Visible at a glance alongside the provider table so the
        // User doesn't forget why the resolver is picking one over
        // The other.
        const result = await isolated('auth status with default')
            .exec(['auth default anthropic', 'auth'])
            .run();

        expect(result.exitCode).toBe(0);
        const err = result.stderr.text;
        expect(err).toContain('default provider:');
        expect(err).toContain('anthropic');
        expect(err).toContain('Pick a default:');
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
