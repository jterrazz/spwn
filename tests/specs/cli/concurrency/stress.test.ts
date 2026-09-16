import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * CLI stress / concurrency. There is no dedicated concurrency primitive — each
 * spec is a fresh temp dir — so parallel invocations fan out with `Promise.all`
 * over an async helper that binds its own `await using` per exec (rule B5). The
 * lock-held case is docker-aware: a stale per-world lockfile is layered in via
 * the `lock-held/` overlay, and `up` must refuse without ever creating a
 * container — an assertion on the CONTAINER accessor, which is why it stays a
 * chain. The two sequential specs beside them are documents.
 */
/*
 * Create one agent in a fresh isolated home, binding its own `await using` so
 * the fan-out below stays B5-compliant. Module-scoped so it isn't recreated per
 * call; returns the captured outcome for the test to assert on.
 */
async function createInIsolatedHome(name: string): Promise<{ exitCode: number; stderr: string }> {
    await using result = await cli
        .fixture('$FIXTURES/empty/')
        .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
        .exec(['init', `agent create ${name}`]);
    return { exitCode: result.exitCode, stderr: result.stderr.text };
}

describe('cli stress tests', () => {
    test('5 parallel agent create commands all exit cleanly', async () => {
        // Given - five distinct agent names, each run in its own fresh isolated home
        const names = ['alpha', 'bravo', 'charlie', 'delta', 'echo'];

        // Then - every parallel run exits zero with no panic (scalpel: crash-signal absence)
        const outcomes = await Promise.all(
            names.map(async (name) => await createInIsolatedHome(name)),
        );
        for (const outcome of outcomes) {
            expect(outcome.exitCode).toBe(0);
            expect(outcome.stderr).not.toContain('panic:');
            expect(outcome.stderr).not.toContain('goroutine ');
        }
    });

    test('spwn up refuses to run when another up is holding the lock', async () => {
        // Given - a stale per-world lockfile already present under .spwn/ (a concurrent up that has not released)
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .fixture('lock-held/')
            .exec('up neo');

        // Then - exits non-zero with the in-progress message and no container was created (scalpel: error wording + absence)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toContain('Up in progress');
        expect(result.container('neo').exists).toBe(false);
    });
});
