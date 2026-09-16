import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn auth` — the credentials dashboard itself. Every row it prints
 * depends on the HOST: the macOS keychain, the provider variables in the
 * operator's shell, whether a tool is authenticated at all. A byte-exact
 * `stderr:` would pin one machine's credential state and fail on every
 * other, so the dashboard stays a set of intent-level substring probes —
 * the one shape rule D11's full-golden preference makes an exception for.
 *
 * The deterministic half of the command — the retired subcommands, the
 * default-provider verbs, the login help page and its unknown-provider
 * refusal — is a document beside this file.
 *
 * Every result runs with an isolated SPWN_HOME so the real keychain is
 * never written to, and binds with `await using` (rule B5); these are
 * hermetic and CLI-only.
 */

const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

/** A fresh home with the keychain skipped, so nothing reads as authenticated. */
const withoutKeychain = () =>
    cli.fixture('$FIXTURES/empty/').env({
        HOME: '$WORKDIR/empty-home',
        SPWN_HOME: '$WORKDIR/spwn-home',
        SPWN_SKIP_KEYCHAIN: '1',
    });

describe('cli - auth dashboard', () => {
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
        await using result = await withoutKeychain().exec('auth');

        // Then - the MCP section header, a provider row, and its login hint are present
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Tools (MCP)');
        expect(result.stderr).toContain('notion');
        expect(result.stderr).toContain('spwn auth login notion');
    });

    test("'spwn auth' surfaces the CLI-tools section (github)", async () => {
        // Given - a fresh home with keychain skipped
        await using result = await withoutKeychain().exec('auth');

        // Then - the CLI-tools section advertises the github login path
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Tools (CLI)');
        expect(result.stderr).toContain('github');
        expect(result.stderr).toContain('spwn auth login github');
    });

    test("'spwn auth' lists every supported method per provider", async () => {
        // Given - a fresh home with no host creds, keychain skipped
        await using result = await withoutKeychain().exec('auth');

        // Then - each unset row names the exact command to set that provider/method (scalpel: hint text iterates)
        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/claude login/);
        expect(stderr).toMatch(/codex login|OPENAI_API_KEY/);
        expect(stderr).toMatch(/spwn auth login anthropic --api-key/);
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
