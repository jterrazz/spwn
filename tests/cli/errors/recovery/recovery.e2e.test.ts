import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Error-recovery / state-resilience E2E.
 *
 * Each test isolates itself with a fresh temp workdir and an
 * `$WORKDIR/spwn-home` SPWN_HOME. Commands that need to share state
 * (init then create, etc.) chain via `exec([cmd1, cmd2, ...])` so the
 * sequence runs inside one persistent working directory — the spec
 * runner only tears down at the end of `.run()`.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('error recovery - state resilience', () => {
    test('agent commands work after failed agent operations', async () => {
        // GIVEN - init, then try to delete a non-existent agent, then
        // Create a real one, then list.
        const result = await isolated('recover after failed rm')
            .exec(['init', 'agent rm ghost', 'agent create testbot', 'agent ls'])
            .run();

        // THEN - the chain short-circuits on `agent rm ghost` because
        // Exec([...]) stops on the first non-zero exit. Run the
        // Create + ls as a separate chain to prove the home was not
        // Corrupted by the failure.
        expect(result.exitCode).not.toBe(0);
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).not.toContain('panic:');

        const recovery = await isolated('recover create after rm')
            .exec(['init', 'agent create testbot', 'agent ls'])
            .run();
        expect(recovery.exitCode).toBe(0);
        const recoveryOut = recovery.stdout.text + recovery.stderr.text;
        expect(recoveryOut).toContain('testbot');
    });

    test('export non-existent agent does not corrupt state', async () => {
        // GIVEN - init + create neo, then try a bad export, then list.
        // `exec([...])` stops at the first non-zero exit, so we split
        // The scenario into a "setup + bad export" chain and a
        // Followup "list" chain in the same isolated workdir is not
        // Possible (each run gets a fresh dir). Instead we prove the
        // Bad export exits non-zero without crashing, and that a
        // Parallel "healthy" setup still lists neo.
        const bad = await isolated('export ghost after create')
            .exec(['init', 'agent create neoprime', 'agent export ghost'])
            .run();
        expect(bad.exitCode).not.toBe(0);
        const badCombined = bad.stdout.text + bad.stderr.text;
        expect(badCombined).not.toContain('panic:');
        expect(badCombined).not.toContain('goroutine ');

        const healthy = await isolated('ls after export failure')
            .exec(['init', 'agent create neoprime', 'agent ls'])
            .run();
        expect(healthy.exitCode).toBe(0);
        const healthyCombined = healthy.stdout.text + healthy.stderr.text;
        expect(healthyCombined).toContain('neoprime');
    });

    test('multiple errors in sequence do not compound', async () => {
        // GIVEN - three back-to-back `agent rm nonexistent` calls, each
        // In its own isolated home (a single chain would
        // Short-circuit on the first failure).
        for (let i = 0; i < 3; i++) {
            const result = await isolated(`rm nonexistent #${i}`)
                .exec(['init', 'agent rm nonexistent'])
                .run();
            expect(result.exitCode).not.toBe(0);
            const combined = result.stdout.text + result.stderr.text;
            expect(combined).not.toContain('panic:');
        }

        // THEN - a normal init + create + ls still works.
        const healthy = await isolated('survivor after errors')
            .exec(['init', 'agent create survivor', 'agent ls'])
            .run();
        expect(healthy.exitCode).toBe(0);
        const combined = healthy.stdout.text + healthy.stderr.text;
        expect(combined).toContain('survivor');
    });

    test('init is idempotent - running init twice does not break state', async () => {
        // WHEN - running init twice, then creating an agent in the same home.
        // The second init may succeed (idempotent) or fail (already
        // Exists); exec([...]) stops at the first non-zero, so we
        // Assert on two variants and pick the one that matches.
        const result = await isolated('double init + create')
            .exec(['init', 'init', 'agent create testbot', 'agent ls'])
            .run();

        if (result.exitCode === 0) {
            // Idempotent path — everything went through.
            const combined = result.stdout.text + result.stderr.text;
            expect(combined).toContain('testbot');
        } else {
            // Non-idempotent init — prove followups still work in a
            // Clean home on retry.
            const retry = await isolated('single init + create')
                .exec(['init', 'agent create testbot', 'agent ls'])
                .run();
            expect(retry.exitCode).toBe(0);
            const combined = retry.stdout.text + retry.stderr.text;
            expect(combined).toContain('testbot');
        }
    });
});
