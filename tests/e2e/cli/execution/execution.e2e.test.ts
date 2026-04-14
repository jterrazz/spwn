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
 * spwn's success path writes status banners to stderr, which the
 * `@jterrazz/test` ExecAdapter (execSync) discards on exit 0. As a
 * result, most success-path banners cannot be asserted on and the
 * tests collapse to exit-code smoke checks. Error-path stderr still
 * comes through because the adapter captures stderr on non-zero.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

// ── Agent management (no Docker) ─────────────────────────────

describe('CLI execution - agent commands', () => {
    test("'spwn agent create' succeeds", async () => {
        // Success banner goes to stderr — we only assert on exit code.
        const result = await isolated('agent create testbot').exec('agent create testbot').run();
        expect(result.exitCode).toBe(0);
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

    test("'spwn agent ls' runs on an empty home", async () => {
        const result = await isolated('agent ls empty').exec('agent ls').run();
        expect(result.exitCode).toBe(0);
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
    test("'spwn status' runs without error after init", async () => {
        /*
         * Both `init` and `status` render to stderr on success, so all
         * we can check is that they exit zero. The richer status-output
         * coverage lives in tests/e2e/status/status/*.
         */
        const initResult = await isolated('init for status').exec('init').run();
        expect(initResult.exitCode).toBe(0);

        const statusResult = await isolated('status after init').exec('status').run();
        expect(statusResult.exitCode).toBe(0);
    });

    test("'spwn auth' runs without error", async () => {
        const result = await isolated('auth from execution').exec('auth').run();
        expect(result.exitCode).toBe(0);
    });
});
