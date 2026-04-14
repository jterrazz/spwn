import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Error-handling E2E — one test per error shape, each paired with a
 * locked-in stderr snapshot under `./expected/stderr/<name>.txt`.
 *
 * spwn writes the "→ Doing X..." / "✗ Failed ..." banners to stderr,
 * which the @jterrazz/test runner captures on both the success and
 * failure paths.
 *
 * Regenerate snapshots with:
 *   JTERRAZZ_TEST_UPDATE=1 pnpm -C tests exec vitest run cli/errors/errors
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('error handling', () => {
    test('destroy non-existent world', async () => {
        const result = await isolated('destroy missing').exec('down w-nonexistent-00000').run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('destroy-missing.txt');
    });

    test('inspect non-existent world', async () => {
        const result = await isolated('inspect missing')
            .exec('world inspect w-nonexistent-00000')
            .run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('inspect-missing.txt');
    });

    test('agent --ephemeral without --world flag', async () => {
        const result = await isolated('npc no world').exec('agent --ephemeral lint-code').run();

        expect(result.exitCode).toBe(1);
    });

    test('agent dream non-existent agent skips gracefully', async () => {
        const result = await isolated('dream missing').exec('agent dream nonexistent').run();

        expect(result.exitCode).toBe(0);
        await result.stderr.toMatch('dream-missing.txt');
    });

    test('agent export non-existent agent', async () => {
        const result = await isolated('export missing').exec('agent export nonexistent').run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('export-missing.txt');
    });

    test('logs for non-existent world', async () => {
        /*
         * `spwn world logs` filters by world ID; a missing world yields
         * no events. The important part is the absence of a crash.
         */
        const result = await isolated('logs missing').exec('world logs w-nonexistent-00000').run();

        expect(result.stderr.text).not.toContain('panic:');
        expect(result.stderr.text).not.toContain('goroutine');
    });

    test('agent talk to non-existent agent', async () => {
        const result = await isolated('talk missing').exec('agent talk nonexistent "hello"').run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('talk-missing.txt');
    });

    test('delete non-existent agent shows error', async () => {
        const result = await isolated('delete ghost').exec('agent rm ghost').run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('delete-missing.txt');
    });

    test('no usage dump on errors', async () => {
        const result = await isolated('error no usage').exec('down w-nonexistent-00000').run();

        expect(result.exitCode).toBe(1);
        // Hygiene: errors must not leak help text into either stream.
        expect(result.stderr.text).not.toContain('Available Commands:');
        expect(result.stderr.text).not.toContain('Global Flags:');
        expect(result.stderr.text).not.toContain('Use "spwn');
        expect(result.stdout.text).not.toContain('Available Commands:');
        expect(result.stdout.text).not.toContain('Global Flags:');
        expect(result.stdout.text).not.toContain('Use "spwn');
    });

    test('error messages follow the structured ✗ convention', async () => {
        /*
         * Redundant with "destroy non-existent world" but cheaper than
         * maintaining a separate assertion on lowercase/format: we
         * reuse the same snapshot to anchor the wording.
         */
        const result = await isolated('error format check').exec('down w-nonexistent-00000').run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('destroy-missing.txt');
    });
});
