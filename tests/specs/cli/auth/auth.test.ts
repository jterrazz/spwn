import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn auth` — credentials dashboard flows. Every test runs with an isolated
 * SPWN_HOME so the real keychain is never touched. `spwn auth` renders to
 * stderr (Unix: data on stdout, status on stderr); row content varies with
 * host state (keychain on macOS, env on CI), so these stay intent-level
 * substring probes rather than goldens. Every result binds with `await using`
 * (rule B5); these are hermetic and CLI-only.
 */

const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('cli - auth command', () => {
    test("'spwn auth' renders the credentials dashboard", async () => {
        // Given - an isolated home
        await using result = await isolated().exec('auth');

        // Then - the stable scaffolding renders (scalpel: row content is host-dependent)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Credentials');
        expect(result.stderr).toContain('Anthropic');
        expect(result.stderr).toContain('OpenAI');
        expect(result.stderr).toContain('oauth');
        expect(result.stderr).toContain('api_key');
        expect(result.stderr).toContain('Default:');
    });

    test("'spwn auth' surfaces the MCP-tools section", async () => {
        // Given - a fresh home with keychain skipped so nothing is authenticated
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .env({
                HOME: '$WORKDIR/empty-home',
                SPWN_HOME: '$WORKDIR/spwn-home',
                SPWN_SKIP_KEYCHAIN: '1',
            })
            .exec('auth');

        // Then - the MCP section header, a provider row, and its login hint are present
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Tools (MCP)');
        expect(result.stderr).toContain('notion');
        expect(result.stderr).toContain('spwn auth login notion');
    });

    test("'spwn auth' surfaces the CLI-tools section (github)", async () => {
        // Given - a fresh home with keychain skipped
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .env({
                HOME: '$WORKDIR/empty-home',
                SPWN_HOME: '$WORKDIR/spwn-home',
                SPWN_SKIP_KEYCHAIN: '1',
            })
            .exec('auth');

        // Then - the CLI-tools section advertises the github login path
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Tools (CLI)');
        expect(result.stderr).toContain('github');
        expect(result.stderr).toContain('spwn auth login github');
    });

    test("'spwn auth' lists every supported method per provider", async () => {
        // Given - a fresh home with no host creds, keychain skipped
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .env({
                HOME: '$WORKDIR/empty-home',
                SPWN_HOME: '$WORKDIR/spwn-home',
                SPWN_SKIP_KEYCHAIN: '1',
            })
            .exec('auth');

        // Then - each unset row names the exact command to set that provider/method (scalpel: hint text iterates)
        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/claude login/);
        expect(stderr).toMatch(/codex login|OPENAI_API_KEY/);
        expect(stderr).toMatch(/spwn auth login anthropic --api-key/);
    });

    test("'spwn auth status' returns a clean error (command retired)", async () => {
        // Given - the retired `auth status` subcommand
        await using result = await isolated().exec('auth status');

        // Then - a crisp unknown-command error (error-message probe)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toContain('unknown command');
    });

    test("'spwn auth check' returns a clean error (command retired)", async () => {
        // Given - the retired `auth check` subcommand
        await using result = await isolated().exec('auth check');

        // Then - a crisp unknown-command error (error-message probe)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toContain('unknown command');
    });

    test("'spwn auth token' returns a clean error (command retired)", async () => {
        // Given - the retired `auth token` subcommand
        await using result = await isolated().exec('auth token sk-ant-foo');

        // Then - a crisp unknown-command error (error-message probe)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toContain('unknown command');
    });

    test("'spwn auth --help' shows the remaining subcommand surface", async () => {
        // Given - the auth help page
        await using result = await isolated().exec('auth --help');

        // Then - the live verbs are listed and the retired ones are gone (cobra help probe)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('auth');
        expect(result.stdout).toContain('login');
        expect(result.stdout).toContain('logout');
        expect(result.stdout).toContain('use');
        expect(result.stdout).toContain('default');
        expect(result.stdout).not.toContain('status');
        expect(result.stdout).not.toContain('check');
    });

    test("'spwn auth default' on a fresh home reports 'not set'", async () => {
        // Given - a fresh home with no default provider
        await using result = await isolated().exec('auth default');

        // Then - the hint guides the user to the setter (scalpel: dashboard status text)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('default provider:');
        expect(result.stderr).toContain('not set');
        expect(result.stderr).toContain('spwn auth default <provider>');
    });

    test("'spwn auth default <provider>' sets the preference", async () => {
        // Given - setting anthropic as the default provider
        await using result = await isolated().exec('auth default anthropic');

        // Then - the banner confirms and auth.yaml carries the field
        expect(result.exitCode, result.stderr.text).toBe(0);
        expect(result.stderr).toContain('default provider set to anthropic');
        expect(result.file('spwn-home/auth.yaml').content).toContain('default_provider: anthropic');
    });

    test("'spwn auth default' reads back what was previously set", async () => {
        // Given - a set then a read in one chain sharing the workdir (and SPWN_HOME)
        await using result = await isolated().exec(['auth default anthropic', 'auth default']);

        // Then - the read reports the persisted value plus the clear hint
        expect(result.exitCode, result.stderr.text).toBe(0);
        expect(result.stderr).toContain('default provider:');
        expect(result.stderr).toContain('anthropic');
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/Clear with:\s+spwn auth default --clear/);
    });

    test("'spwn auth default --clear' unsets the preference", async () => {
        // Given - a set then a clear in one chain
        await using result = await isolated().exec(['auth default openai', 'auth default --clear']);

        // Then - the banner confirms and the field is gone from auth.yaml
        expect(result.exitCode, result.stderr.text).toBe(0);
        expect(result.stderr).toContain('default provider cleared');
        expect(result.file('spwn-home/auth.yaml').content).not.toMatch(
            /default_provider:\s*openai/,
        );
    });

    test("'spwn auth default' refuses a disabled provider", async () => {
        // Given - a disabled provider then an attempt to default to it
        await using result = await isolated().exec(['auth disable openai', 'auth default openai']);

        // Then - a loud error beats a subtle no-op (error-message probe)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toContain('is currently disabled');
        expect(result.stderr).toContain('spwn auth enable openai');
    });

    test("'spwn auth' dashboard shows the default provider when set", async () => {
        // Given - a default set then the dashboard rendered in one chain
        await using result = await isolated().exec(['auth default anthropic', 'auth']);

        // Then - the dashboard surfaces the default at a glance
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Default:');
        expect(result.stderr).toContain('anthropic');
    });

    /*
     * SKIPPED: `spwn auth login` enters an interactive prompt reading from
     * stdin; the exec adapter has no way to pipe empty input / EOF reliably
     * without hanging.
     */
    test.todo("'spwn auth login' handles non-interactive gracefully");

    test("'spwn auth login --help' documents the MCP-provider path", async () => {
        // Given - the auth login help page
        await using result = await isolated().exec('auth login --help');

        // Then - the MCP block advertises `spwn auth login notion` (cobra help probe)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('hosted MCP providers');
        expect(result.stdout).toContain('Notion');
        expect(result.stdout).toContain('spwn auth login notion');
    });

    test("'spwn auth login bogus' lists both AI and MCP providers", async () => {
        // Given - an unknown provider name
        await using result = await isolated().exec('auth login bogus-name');

        // Then - the error surfaces both registries (error-message probe)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('unknown provider');
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/anthropic|openai/);
        expect(result.stderr).toContain('MCP providers');
        expect(result.stderr).toContain('notion');
    });

    test("'spwn auth login notion' takes the MCP branch (not the API-key branch)", async () => {
        // Given - a dead DOCKER_HOST so login fails fast at the helper-image build
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .env({
                DOCKER_HOST: 'tcp://127.0.0.1:1',
                SPWN_HOME: '$WORKDIR/spwn-home',
            })
            .exec('auth login notion');

        // Then - the failure references the MCP/helper path, proving we did not fall to the API-key branch
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text.toLowerCase()).toMatch(/notion|helper|docker|mcp|oauth|build/);
        expect(
            result.file('spwn-home/credentials/mcp/oauth').exists ||
                result.file('spwn-home/credentials/mcp/oauth/').exists,
        ).toBe(false);
    });

    /*
     * SKIPPED: `spwn auth logout <provider>` reads and mutates the anthropic
     * credential from the OS keychain, which SPWN_HOME isolation does not stub.
     * Running it against the real keychain deletes the user's Claude Code OAuth
     * token. Re-enable once the auth layer grows a keychain-stub mode.
     */
    test.todo("'spwn auth logout <provider>' on a fresh home emits the no-op banner");

    test.todo("'spwn auth logout <provider>' is idempotent");
});
