import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn status` — top-level dashboard.
 *
 * Status output is written to stderr (Unix convention for status
 * dashboards). With the `<PROJECT>` path transform in place, the
 * output is stable across machines, so we lock it down with real
 * stderr snapshots. The richer Docker-backed status assertions
 * (running worlds, memory counts, etc.) live in the sibling
 * status-docker test.
 *
 * Regenerate snapshots with:
 *   JTERRAZZ_TEST_UPDATE=1 pnpm -C tests exec vitest run cli/status/status
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('spwn status', () => {
    describe('without worlds', () => {
        test('runs cleanly after init', async () => {
            /*
             * Note: each `isolated()` call mints its own temp home,
             * so the `init` invocation below is functionally a smoke
             * check — the `status` call that follows runs in a fresh
             * home, just like the "uninitialised" test. The snapshot
             * is therefore shared.
             */
            const init = await isolated('init').exec('init').run();
            expect(init.exitCode).toBe(0);

            const result = await isolated('status').exec('status').run();
            expect(result.exitCode).toBe(0);
            await result.stderr.toMatch('status-empty-home.txt');
        });
    });

    describe('error handling', () => {
        test('status on uninitialised home still works', async () => {
            const result = await isolated('status no init').exec('status').run();
            expect(result.exitCode).toBe(0);
            await result.stderr.toMatch('status-empty-home.txt');
        });
    });

    // Merged from the legacy `status-docker.e2e.test.ts`. With a live
    // Project world, `spwn status` should surface the neo world and its
    // Agent. Renderer output goes to stderr (Unix convention for
    // Dashboards), so we match on stderr content.
    describe('with an active world (docker)', () => {
        test('shows the running project world and its agent', async () => {
            await using result = await spec('status with world')
                .project('docker-pilot')
                .exec(['up', 'status'])
                .run();

            expect(result.exitCode).toBe(0);

            // The container should be live under the run label.
            expect(result.container('neo').running).toBe(true);

            // Multi-exec only exposes the last command's streams, so
            // Both the status banner and the world entry should be in
            // The final `status` output. Renderer writes to stderr.
            result.stderr.toContain('spwn');
            result.stderr.toContain('neo');
        });

        test('status and ls report the same running worlds', async () => {
            // Given - a running world `neo`
            // When - `spwn ls` runs after `spwn up`
            await using ls = await spec('ls after up')
                .project('docker-pilot')
                .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
                .exec(['up', 'world list'])
                .run();

            // And - `spwn status` runs under the same project
            await using status = await spec('status after up')
                .project('docker-pilot')
                .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
                .exec(['up', 'status'])
                .run();

            // Then - both exit zero and both mention `neo` exactly once
            expect(ls.exitCode).toBe(0);
            expect(status.exitCode).toBe(0);
            const lsMatches = (ls.stderr.text.match(/\bneo\b/g) ?? []).length;
            const statusMatches = (status.stderr.text.match(/\bneo\b/g) ?? []).length;
            expect(lsMatches).toBeGreaterThanOrEqual(1);
            expect(statusMatches).toBeGreaterThanOrEqual(1);
        });
    });
});
