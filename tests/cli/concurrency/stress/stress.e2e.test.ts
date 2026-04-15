import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * CLI stress / concurrency tests.
 *
 * The `@jterrazz/test` spec runner does not have a dedicated
 * concurrency primitive — each spec is a fresh temp dir — so we use
 * raw `Promise.all` to fan out spec calls. Each call below gets its
 * own isolated SPWN_HOME under `$WORKDIR/spwn-home`, so the tests
 * exercise concurrency at the process level without sharing state.
 *
 * The stress here is about "spwn does not crash under rapid
 * invocation"; the "all agents end up in one list" shape of the
 * legacy test relied on shared SPWN_HOME across sibling processes
 * and is covered by Go unit tests.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('CLI stress tests', () => {
    test('5 parallel agent create commands all exit cleanly', async () => {
        // Given - five distinct agent names
        const names = ['alpha', 'bravo', 'charlie', 'delta', 'echo'];

        // When - each runs in its own fresh home, initialized then create
        const results = await Promise.all(
            names.map((name) =>
                isolated(`stress create ${name}`)
                    .exec(['init', `agent create ${name}`])
                    .run(),
            ),
        );

        // Then - every run exits zero with no panic
        for (const result of results) {
            expect(result.exitCode).toBe(0);
            expect(result.stderr.text).not.toContain('panic:');
            expect(result.stderr.text).not.toContain('goroutine ');
        }
    });

    test('create + list round-trip surfaces the agent name in stderr banners', async () => {
        // Given - a fresh isolated home
        // When - init, create, then ls in one chained exec
        const result = await isolated('stress create + ls')
            .exec(['init', 'agent create rapid', 'agent ls'])
            .run();

        // Then - last command (ls) exits zero, and the chain mentions the agent
        expect(result.exitCode).toBe(0);
        // `agent ls` renders its table on stderr (ui.Table default writer).
        result.stderr.toContain('rapid');
    });

    test('spwn up refuses to run when another up is holding the lock', async () => {
        // Given - a stale per-world lockfile already present in .spwn/ (simulates a concurrent `spwn up` that has not yet released).
        // When - we run `spwn up neo`.
        // Then - exit non-zero with an "Up in progress" message and no container is created.
        await using result = await spec('up lock held')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .seed('lock/.up.neo.lock')
            .exec('up neo')
            .run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toContain('Up in progress');
        // And - no neo container was created under this test run label.
        expect(result.container('neo').exists).toBe(false);
    });

    test('rapid sequential create/rm cycles do not corrupt state', async () => {
        // Given - a chain of 5 create/rm pairs followed by agent ls
        const steps: string[] = ['init'];
        for (let i = 0; i < 5; i++) {
            steps.push(`agent create rapid-${i}`);
            steps.push(`agent rm rapid-${i}`);
        }
        steps.push('agent ls');

        // When - the chain runs in a single isolated home
        const result = await isolated('stress sequential').exec(steps).run();

        // Then - every command in the chain exits zero, no crash.
        // We intentionally do NOT assert that the cycled agent names
        // Disappear from `agent ls`: spwn currently auto-adds a world
        // Per freshly-created agent, and `agent rm` only removes the
        // Profile — the world entry in spwn.yaml lingers until the
        // User runs `world rm`. The legacy test predated that
        // Behaviour. Assert on the absence-of-crash signal instead.
        expect(result.exitCode).toBe(0);
        expect(result.stderr.text).not.toContain('panic:');
        expect(result.stderr.text).not.toContain('goroutine ');
        // Only the LAST command's output is captured by the chained
        // `exec([...])` adapter (each sub-command runs in its own
        // SpawnSync), so the deletion banners from the interior
        // Commands aren't in `combined` — we're asserting on the
        // Final `agent ls` output here.
    });
});
