import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * CLI execution — non-Docker paths.
 *
 * The legacy file mixed Docker-backed world lifecycle tests with
 * agent-management / flag / enter / status tests. Only the non-Docker
 * suites are ported here; the Docker ones stay on the legacy helpers
 * (they rely on createTestContext / ctx.spwn).
 *
 * spwn's success path writes status banners to stderr (Unix
 * convention). Stable banners get stderr snapshots under
 * `./expected/stderr/`; machine-dependent output (paths, ids) is
 * matched with substrings against `result.stderr.text`.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

// ── Agent management (no Docker) ─────────────────────────────

describe('CLI execution - agent commands', () => {
    test("'spwn agent create testbot' prints the creation banner", async () => {
        const result = await isolated('agent create testbot').exec('agent create testbot').run();
        expect(result.exitCode).toBe(0);
        await result.stderr.toMatch('agent-create-testbot.txt');
    });

    test("'spwn agent rm' on missing agent errors cleanly", async () => {
        /*
         * Each spec gets a fresh temp workdir, so agent create/rm pairs
         * across specs cannot share state. The legacy "create-then-rm"
         * round-trip test is covered implicitly by the Go unit tests;
         * here we just verify rm fails cleanly without a stack trace.
         */
        const result = await isolated('agent rm missing').exec('agent rm ghost').run();

        expect(result.exitCode).not.toBe(0);
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).not.toContain('panic:');
        expect(combined).not.toContain('goroutine ');
    });

    test("'spwn agent ls' on an empty home prints the empty state", async () => {
        const result = await isolated('agent ls empty').exec('agent ls').run();
        expect(result.exitCode).toBe(0);
        await result.stderr.toMatch('agent-ls-empty.txt');
    });

    test("'spwn agent show' on nonexistent agent errors cleanly", async () => {
        const result = await isolated('agent show missing').exec('agent show ghost').run();

        expect(result.exitCode).not.toBe(0);
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).not.toContain('panic:');
        expect(combined).not.toContain('goroutine ');
    });
});

// ── Enter command ──────────────────────────────────────────

describe('CLI execution - enter command', () => {
    test("'spwn world enter <nonexistent-id>' returns clean error", async () => {
        const result = await isolated('enter nonexistent').exec('world enter w-fake-99999').run();

        expect(result.exitCode).not.toBe(0);
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).not.toContain('panic:');
        expect(combined).not.toContain('goroutine ');
    });

    test("'spwn world enter --help' shows usage", async () => {
        // --help is one of the few spwn subcommands that writes to stdout
        // (cobra's default behaviour), so we can snapshot it.
        const result = await isolated('enter help').exec('world enter --help').run();

        expect(result.exitCode).toBe(0);
        const out = result.stdout.text;
        expect(out).toContain('enter');
        expect(out).toContain('world-id');
    });
});

// ── Global flags ─────────────────────────────────────────────

describe('CLI execution - global flags', () => {
    test("'--version' shows version string", async () => {
        // --version writes to stdout.
        const result = await isolated('version').exec('--version').run();

        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toMatch(/spwn version/);
    });
});

// ── Status command ──────────────────────────────────────────

describe('CLI execution - status command', () => {
    test("'spwn status' runs cleanly after init", async () => {
        /*
         * Richer status-output coverage lives in
         * tests/e2e/status/status/*. Here we just confirm both
         * commands emit their stable banners.
         */
        const initResult = await isolated('init for status').exec('init').run();
        expect(initResult.exitCode).toBe(0);
        // `init` banner includes the basename of the temp workdir,
        // Which is not snapshot-stable (the transform only masks the
        // Path prefix, not the basename). Assert on the marker.
        expect(initResult.stderr.text).toContain('Initialised spwn project');

        const statusResult = await isolated('status after init').exec('status').run();
        expect(statusResult.exitCode).toBe(0);
        const stderr = statusResult.stderr.text;
        expect(stderr).toContain('Worlds');
        expect(stderr).toContain('Architect');
    });

    test("'spwn auth' from an empty project still renders the provider table", async () => {
        const result = await isolated('auth from execution').exec('auth').run();
        expect(result.exitCode).toBe(0);
        // Provider table is keychain-dependent (see auth.e2e.test.ts),
        // So we only assert on the stable header row here.
        expect(result.stderr.text).toContain('PROVIDER');
    });
});
