import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Spwn auth — credentials dashboard flows.
 *
 * Every test runs with an isolated SPWN_HOME so we never touch the
 * User's real keychain or credentials file. No real network calls
 * Are made on a fresh home (no credentials → no validation targets),
 * Making the dashboard output stable.
 *
 * `spwn auth` renders to stderr (Unix: data on stdout, status on
 * Stderr). Content varies with host state (keychain on macOS, env
 * Vars on CI), so we assert on intent-level substrings rather than
 * Snapshotting.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('CLI - auth command', () => {
    test("'spwn auth' renders the credentials dashboard", async () => {
        const result = await isolated('auth dashboard').exec('auth').run();

        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        // Hero + provider blocks + default footer are the stable
        // Scaffolding. Content of each row varies by host state.
        expect(stderr).toContain('Credentials');
        expect(stderr).toContain('Anthropic');
        expect(stderr).toContain('OpenAI');
        expect(stderr).toContain('oauth');
        expect(stderr).toContain('api_key');
        expect(stderr).toContain('Default:');
    });

    test("'spwn auth' surfaces the MCP-tools section", async () => {
        // Discoverability: the only place a user finds out that
        // `spwn auth login notion` exists is the dashboard. The
        // Section header + at least one provider row + the per-row
        // Hint must all be there.
        const result = await isolated('auth dashboard mcp section')
            .env({
                SPWN_HOME: '$WORKDIR/spwn-home',
                HOME: '$WORKDIR/empty-home',
                SPWN_SKIP_KEYCHAIN: '1',
            })
            .exec('auth')
            .run();
        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toContain('Tools (MCP)');
        expect(stderr).toContain('notion');
        // Fresh home → not authenticated → row points at the login verb
        expect(stderr).toContain('spwn auth login notion');
    });

    test("'spwn auth' surfaces the CLI-tools section (github)", async () => {
        // Same discoverability concern as the MCP section — the
        // Dashboard must advertise `spwn auth login github` so
        // Users find the gh-import path without reading code.
        const result = await isolated('auth dashboard cli section')
            .env({
                SPWN_HOME: '$WORKDIR/spwn-home',
                HOME: '$WORKDIR/empty-home',
                SPWN_SKIP_KEYCHAIN: '1',
            })
            .exec('auth')
            .run();
        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toContain('Tools (CLI)');
        expect(stderr).toContain('github');
        expect(stderr).toContain('spwn auth login github');
    });

    // The hermetic "host gh missing" + "host gh logged out"
    // Failure-mode tests live in Go (packages/auth/gh/login_test.go)
    // Because they need PATH manipulation that the TS harness
    // Doesn't isolate from the parent process's PATH. The TS
    // Dashboard tests above guarantee discoverability — together
    // They cover the surface.

    test("'spwn auth' lists every supported method per provider", async () => {
        // Fresh home with no host creds → every method is unset, so
        // The dashboard doubles as a cheat-sheet: each row must name
        // The exact command to set that provider/method combo.
        // SPWN_SKIP_KEYCHAIN + HOME override prevents leaking the dev
        // Box's real claude login into the test.
        const result = await spec('auth method catalog')
            .project('empty')
            .env({
                SPWN_HOME: '$WORKDIR/spwn-home',
                HOME: '$WORKDIR/empty-home',
                SPWN_SKIP_KEYCHAIN: '1',
            })
            .exec('auth')
            .run();

        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        // Unset rows point at the fix command. Use generic matches
        // Since the exact hint text is subject to iteration.
        expect(stderr).toMatch(/claude login/);
        expect(stderr).toMatch(/codex login|OPENAI_API_KEY/);
        expect(stderr).toMatch(/spwn auth login anthropic --api-key/);
    });

    test("'spwn auth status' returns a clean error (command retired)", async () => {
        // Users with muscle memory for the deleted subcommands get a
        // Crisp "unknown command" so they read the help and discover
        // The new bare-command UX.
        const result = await isolated('retired status').exec('auth status').run();
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toContain('unknown command');
    });

    test("'spwn auth check' returns a clean error (command retired)", async () => {
        const result = await isolated('retired check').exec('auth check').run();
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toContain('unknown command');
    });

    test("'spwn auth token' returns a clean error (command retired)", async () => {
        const result = await isolated('retired token').exec('auth token sk-ant-foo').run();
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toContain('unknown command');
    });

    test("'spwn auth --help' shows the remaining subcommand surface", async () => {
        const result = await isolated('auth help').exec('auth --help').run();

        expect(result.exitCode).toBe(0);
        const out = result.stdout.text;
        expect(out).toContain('auth');
        expect(out).toContain('login');
        expect(out).toContain('logout');
        expect(out).toContain('use');
        expect(out).toContain('default');
        // Retired verbs must not show up in the help surface.
        expect(out).not.toContain('status');
        expect(out).not.toContain('check');
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

    test("'spwn auth' dashboard shows the default provider when set", async () => {
        // Dashboard contract: once a default exists, it must be
        // Visible at a glance so the user doesn't forget why the
        // Resolver picks one over the other when both are present.
        const result = await isolated('auth dashboard with default')
            .exec(['auth default anthropic', 'auth'])
            .run();

        expect(result.exitCode).toBe(0);
        const err = result.stderr.text;
        expect(err).toContain('Default:');
        expect(err).toContain('anthropic');
    });

    /*
     * SKIPPED: `spwn auth login` enters an interactive prompt reading
     * from stdin; the ExecAdapter has no way to pipe empty input / EOF
     * reliably without hanging. See tests/setup/spwn.specification.ts.
     */
    test.skip("'spwn auth login' handles non-interactive gracefully", () => {});

    test("'spwn auth login --help' documents the MCP-provider path", async () => {
        // The MCP block is what tells users `spwn auth login notion`
        // Is even a thing — it's the only discovery surface short of
        // Reading the provider registry. Drift here = users don't
        // Find it.
        const result = await isolated('login help mcp').exec('auth login --help').run();
        expect(result.exitCode).toBe(0);
        const out = result.stdout.text;
        expect(out).toContain('hosted MCP providers');
        expect(out).toContain('Notion');
        expect(out).toContain('spwn auth login notion');
    });

    test("'spwn auth login bogus' lists both AI and MCP providers", async () => {
        // The error must surface both registries — a user typing a
        // Wrong name shouldn't have to know which family the right
        // Name lives in.
        const result = await isolated('login unknown provider').exec('auth login bogus-name').run();
        expect(result.exitCode).not.toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toContain('unknown provider');
        // AI providers from parseProvider's error
        expect(stderr).toMatch(/anthropic|openai/);
        // MCP providers from the wrap
        expect(stderr).toContain('MCP providers');
        expect(stderr).toContain('notion');
    });

    test("'spwn auth login notion' takes the MCP branch (not the API-key branch)", async () => {
        // Without docker, login fails fast at ensureHelperImage. The
        // Shape of the failure tells us we hit the MCP path: the
        // Error mentions the helper image, not API keys. We point
        // DOCKER_HOST at a dead address so this test stays
        // Hermetic — no docker daemon required.
        const result = await isolated('login mcp branch')
            .env({
                DOCKER_HOST: 'tcp://127.0.0.1:1',
                SPWN_HOME: '$WORKDIR/spwn-home',
            })
            .exec('auth login notion')
            .run();
        expect(result.exitCode).not.toBe(0);
        const stderr = result.stderr.text;
        // The error message must reference the helper image build /
        // Mcp oauth path — proof we did NOT silently fall through
        // To the API-key dashboard.
        expect(stderr.toLowerCase()).toMatch(/notion|helper|docker|mcp|oauth|build/);
        // The cache dir gets created (it's a precondition for the
        // Bind-mount), but no tokens.json should have landed for
        // Any provider — every login path past dir-create must
        // Fail with no docker.
        expect(
            result.file('spwn-home/credentials/mcp/oauth').exists ||
                result.file('spwn-home/credentials/mcp/oauth/').exists,
        ).toBe(false);
    });

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
