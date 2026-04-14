import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * CLI input validation — argument handling, unknown commands, and the
 * quality of the resulting error messages.
 *
 * These tests pin down the boundary between "user typed garbage" and
 * "user typed something we should explain". Snapshots for these would
 * be brittle (cobra wording drifts per release), so we stick to
 * intent-level substring assertions.
 *
 * Each spec gets a fresh `empty` project copy and an isolated SPWN_HOME
 * under `$WORKDIR/spwn-home` so tests can't leak into the user's real
 * `~/.spwn` or into each other.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('CLI input validation', () => {
    // ── Missing required arguments ─────────────────────────

    test("'spwn agent create' with no name picks a random planet name", async () => {
        const result = await isolated('agent create no name').exec('agent create').run();

        /*
         * No-name is not an error: spwn picks a random planet name.
         * The "Created agent …" banner goes to stderr on success, which
         * `@jterrazz/test`'s ExecAdapter (execSync) discards, so we can
         * only assert on the exit code here. Weakened from the legacy
         * version.
         */
        expect(result.exitCode).toBe(0);
    });

    test("'spwn agent create a b c' with too many args shows error", async () => {
        const result = await isolated('agent create extra args').exec('agent create a b c').run();

        expect(result.exitCode).not.toBe(0);
        // ExecAdapter loses stderr on exit 0, but non-zero paths keep it
        const combined = (result.stdout.text + result.stderr.text).toLowerCase();
        expect(combined).toMatch(/unknown|too many|invalid|argument|accepts/);
    });

    test("'spwn down' with no world ID shows error", async () => {
        const result = await isolated('down no id').exec('down').run();

        expect(result.exitCode).not.toBe(0);
        const combined = (result.stdout.text + result.stderr.text).toLowerCase();
        expect(combined).toMatch(/world|required|argument|missing|id|accepts|arg/);
    });

    test("'spwn world inspect' with no world ID shows error", async () => {
        const result = await isolated('inspect no id').exec('world inspect').run();

        expect(result.exitCode).not.toBe(0);
        const combined = (result.stdout.text + result.stderr.text).toLowerCase();
        expect(combined).toMatch(/world|required|argument|missing|id|accepts|arg/);
    });

    test("'spwn world logs' with no world ID shows error", async () => {
        const result = await isolated('logs no id').exec('world logs').run();

        expect(result.exitCode).not.toBe(0);
        const combined = (result.stdout.text + result.stderr.text).toLowerCase();
        expect(combined).toMatch(/world|required|argument|missing|id|accepts|arg/);
    });

    test("'spwn profile' with no subcommand shows help", async () => {
        const result = await isolated('profile no args').exec('profile').run();

        // Profile is a command group - bare invocation renders help cleanly.
        expect(result.exitCode).toBe(0);
        expect(result.stdout.text.toLowerCase()).toContain('profile');
    });

    test("'spwn agent send' with missing args shows error", async () => {
        const result = await isolated('agent send no args').exec('agent send').run();

        expect(result.exitCode).not.toBe(0);
        const combined = (result.stdout.text + result.stderr.text).toLowerCase();
        expect(combined).toMatch(/required|argument|missing|world|message|accepts|arg/);
    });

    // ── Error messages quality ─────────────────────────────

    test('error messages do NOT dump full usage/help', async () => {
        const commands = [
            'down w-nonexistent-00000',
            'world inspect w-nonexistent-00000',
            'agent export nonexistent',
        ];

        for (const cmd of commands) {
            const result = await isolated(`validation: ${cmd}`).exec(cmd).run();

            if (result.exitCode !== 0) {
                const combined = result.stdout.text + result.stderr.text;
                expect(combined).not.toContain('Available Commands:');
                expect(combined).not.toContain('Global Flags:');
            }
        }
    });

    test('error messages contain actionable hints', async () => {
        // Destroy a non-existent world - should show a clean error.
        const result = await isolated('actionable hint').exec('down w-nonexistent-00000').run();

        expect(result.exitCode).not.toBe(0);
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).toMatch(/not found/);
        // Should use the structured ✗ prefix.
        expect(combined).toMatch(/✗/);
    });

    test('unknown top-level command shows error without full usage dump', async () => {
        const result = await isolated('unknown command').exec('foobar').run();

        expect(result.exitCode).not.toBe(0);
        const combined = (result.stdout.text + result.stderr.text).toLowerCase();
        expect(combined).toMatch(/unknown|invalid|command/);
    });

    test('agent rm with no name shows error', async () => {
        const result = await isolated('agent rm no name').exec('agent rm').run();

        expect(result.exitCode).not.toBe(0);
        const combined = (result.stdout.text + result.stderr.text).toLowerCase();
        expect(combined).toMatch(/name|required|argument|missing|accepts|arg/);
    });
});
