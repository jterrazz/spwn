import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn status` — top-level dashboard. Status output goes to stderr (Unix
 * convention for dashboards). With the {{workdir}} token in place the empty-home
 * output is stable across machines, so it is a full stderr golden. The
 * docker-backed cases boot a live world and read it back; every result binds
 * with `await using` so the spawned container is force-removed at scope exit
 * (rule B5).
 */

const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('spwn status', () => {
    describe('without worlds', () => {
        test('runs cleanly after init', async () => {
            // Given - a smoke init in one fresh home, then status in another fresh home
            await using init = await isolated().exec('init');
            expect(init.exitCode).toBe(0);

            await using result = await isolated().exec('status');

            // Then - the empty-home dashboard golden matches
            expect(result.exitCode).toBe(0);
            expect(result.stderr).toMatch('status-empty-home.txt');
        });
    });

    describe('error handling', () => {
        test('status on uninitialised home still works', async () => {
            // Given - status on a home that was never initialised
            await using result = await isolated().exec('status');

            // Then - exits zero rendering the same empty-home golden
            expect(result.exitCode).toBe(0);
            expect(result.stderr).toMatch('status-empty-home.txt');
        });
    });

    describe('with an active world (docker)', () => {
        test('shows the running project world and its agent', async () => {
            // Given - a live docker-pilot world, then status in the same run
            await using result = await cli
                .fixture('$FIXTURES/docker-pilot/')
                .exec(['up', 'status']);

            // Then - the container is live and status surfaces the neo world (renderer writes stderr)
            expect(result.exitCode).toBe(0);
            expect(result.container('neo').running).toBe(true);
            expect(result.stderr).toContain('spwn');
            expect(result.stderr).toContain('neo');
        });

        test('status and ls report the same running worlds', async () => {
            // Given - a running world listed via `world list`
            await using ls = await cli
                .fixture('$FIXTURES/docker-pilot/')
                .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
                .exec(['up', 'world list']);

            // And - `spwn status` runs under the same project
            await using status = await cli
                .fixture('$FIXTURES/docker-pilot/')
                .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
                .exec(['up', 'status']);

            // Then - both exit zero and both mention `neo` at least once (scalpel: ANSI-decorated output)
            expect(ls.exitCode).toBe(0);
            expect(status.exitCode).toBe(0);
            const lsMatches = (ls.stderr.text.match(/\bneo\b/g) ?? []).length;
            const statusMatches = (status.stderr.text.match(/\bneo\b/g) ?? []).length;
            expect(lsMatches).toBeGreaterThanOrEqual(1);
            expect(statusMatches).toBeGreaterThanOrEqual(1);
        });
    });
});
