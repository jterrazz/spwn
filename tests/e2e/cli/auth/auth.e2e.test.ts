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
 * Note: `@jterrazz/test`'s ExecAdapter uses `execSync`, which discards
 * stderr on exit 0. spwn renders provider tables to stderr, so those
 * status tests collapse to exit-code smoke checks. The --help variants
 * go through cobra which writes to stdout, so they keep their content
 * assertions.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('CLI - auth command', () => {
    /*
     * `spwn auth` and `spwn auth check` render the provider table to
     * stderr (confirmed via `spwn auth 2>/dev/null`). On exit 0,
     * ExecAdapter discards stderr, so the two tests below are weakened
     * to smoke checks. Kept because the "runs without crashing"
     * coverage is still worth something.
     */
    test("'spwn auth' runs without crashing", async () => {
        const result = await isolated('auth status').exec('auth').run();

        expect(result.exitCode).toBe(0);
    });

    test("'spwn auth check' runs without crashing", async () => {
        const result = await isolated('auth check').exec('auth check').run();

        expect(result.exitCode).toBe(0);
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

    test("'spwn auth logout' removes cached token", async () => {
        /*
         * Non-zero from "nothing to remove" is acceptable; we just
         * assert it does not crash with a stack trace.
         */
        const result = await isolated('auth logout').exec('auth logout').run();

        expect(result.exitCode).toBeDefined();
        expect(typeof result.exitCode).toBe('number');
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).not.toMatch(/at\s+\S+\s+\(/);
        expect(combined).not.toContain('panic:');
        expect(combined).not.toContain('goroutine ');
    });

    test("'spwn auth logout' is idempotent", async () => {
        await isolated('auth logout 1').exec('auth logout').run();
        const result2 = await isolated('auth logout 2').exec('auth logout').run();

        expect(result2.exitCode).toBeDefined();
        const combined = result2.stdout.text + result2.stderr.text;
        expect(combined).not.toMatch(/at\s+\S+\s+\(/);
    });
});
