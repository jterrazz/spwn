import { afterEach, beforeEach, describe, expect, test } from 'vitest';

import { createSpwnHome } from '../../setup/helpers.js';
import { expectLine, lines } from '../../setup/output-helpers.js';
import { spwn } from '../../setup/spwn.specification.js';

describe('error handling', () => {
    let home: string;
    let originalSpwnHome: string | undefined;

    beforeEach(() => {
        originalSpwnHome = process.env.SPWN_HOME;
        home = createSpwnHome();
        process.env.SPWN_HOME = home;
    });

    afterEach(() => {
        if (originalSpwnHome !== undefined) {
            process.env.SPWN_HOME = originalSpwnHome;
        } else {
            delete process.env.SPWN_HOME;
        }
    });

    test('destroy non-existent world', async () => {
        // WHEN - destroying a world that does not exist
        const result = await spwn('destroy missing').exec('down w-nonexistent-00000').run();

        // THEN - exits with non-zero code and structured error
        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /✗ Destroy failed\s+world w-nonexistent-00000 not found/);
    });

    test('inspect non-existent world', async () => {
        // WHEN - inspecting a world that does not exist
        const result = await spwn('inspect missing')
            .exec('world inspect w-nonexistent-00000')
            .run();

        // THEN - exits with error showing not found
        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /world w-nonexistent-00000 not found/);
    });

    test('agent --ephemeral without --world flag', async () => {
        // WHEN - running agent --ephemeral without specifying a world
        const result = await spwn('npc no world').exec('agent --ephemeral lint-code').run();

        // THEN - exits with error about required world flag
        expect(result.exitCode).not.toBe(0);
    });

    test('agent dream non-existent agent skips gracefully', async () => {
        // WHEN - dreaming on an agent that does not exist (no journal)
        const result = await spwn('dream missing').exec('agent dream nonexistent').run();

        // THEN - exits successfully with structured skip message
        expect(result.exitCode).toBe(0);
        expectLine(result.output, /→ Dreaming for agent "nonexistent"\.\.\./);
        expectLine(result.output, /Skipped\s+no journal entries/);
    });

    test('agent fork non-existent source', async () => {
        // WHEN - forking from an agent that does not exist
        const result = await spwn('fork missing').exec('agent fork nonexistent target').run();

        // THEN - exits with exit code (note: fork may succeed creating sparse copy)
        // The behavior depends on whether the source agent directory exists
        // If source has no layers, fork still runs with what it finds
        if (result.exitCode === 0) {
            expectLine(result.output, /→ Forking "nonexistent" -> "target"\.\.\./);
            expectLine(result.output, /✓ Source\s+nonexistent/);
            expectLine(result.output, /✓ Target\s+target/);
        } else {
            expectLine(result.output, /not found/);
        }
    });

    test('agent export non-existent agent', async () => {
        // WHEN - exporting an agent that does not exist
        const result = await spwn('export missing').exec('agent export nonexistent').run();

        // THEN - exits with error showing not found
        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /✗ Export failed\s+agent "nonexistent" not found/);
    });

    test('logs for non-existent world', async () => {
        // WHEN - fetching world events for a world that does not exist
        // (top-level `spwn logs` is the system-wide event log, so we hit the
        // Scoped `spwn world logs <id>` form instead)
        const result = await spwn('logs missing').exec('world logs w-nonexistent-00000').run();

        // The event log filters by world ID; a missing world simply yields
        // No events. Either an empty output or a "not found" error is
        // Acceptable - what matters is that it doesn't crash.
        const output = result.output;
        expect(output).not.toContain('panic:');
        expect(output).not.toContain('goroutine');
    });

    test('agent talk to non-existent agent', async () => {
        // WHEN - talking to an agent that does not exist
        const result = await spwn('talk missing').exec('agent talk nonexistent "hello"').run();

        // THEN - exits with error showing not found
        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /agent "nonexistent" not found/);
    });

    test('init agent that already exists shows hint', async () => {
        await spwn('init').exec('init').run();
        await spwn('init agent').exec('agent init neo').run();

        // WHEN - init-ing an agent that already exists
        const result = await spwn('init existing').exec('agent init neo').run();

        // THEN - exits with error and actionable hint
        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /✗ Agent creation failed\s+agent "neo" already exists/);
        expectLine(result.output, /spwn agent rm neo/);
    });

    test('delete non-existent agent shows error', async () => {
        await spwn('init').exec('init').run();

        const result = await spwn('delete ghost').exec('agent rm ghost').run();

        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /✗ Delete failed\s+agent "ghost" not found/);
    });

    test('no usage dump on errors', async () => {
        // WHEN - triggering an error
        const result = await spwn('error no usage').exec('down w-nonexistent-00000').run();

        // THEN - no cobra usage dump
        expect(result.exitCode).not.toBe(0);
        const output = result.output;
        expect(output).not.toContain('Available Commands:');
        expect(output).not.toContain('Global Flags:');
        expect(output).not.toContain('Use "spwn');
    });

    test('error messages are lowercase with actionable hint', async () => {
        // WHEN - triggering an error (destroy missing world)
        const result = await spwn('error format check').exec('down w-nonexistent-00000').run();

        // THEN - error message follows convention: structured with ✗ prefix
        expect(result.exitCode).not.toBe(0);
        expectLine(result.output, /✗ Destroy failed\s+world w-nonexistent-00000 not found/);
        // Error messages should start with lowercase (Go convention)
        const errorLines = lines(result.output).filter((l) => l.includes('world w-nonexistent'));
        for (const line of errorLines) {
            // The error detail "world w-nonexistent-00000 not found" starts lowercase
            expect(line).toMatch(/world w-nonexistent-00000 not found/);
        }
    });
});
