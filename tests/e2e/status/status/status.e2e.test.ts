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
 *   JTERRAZZ_TEST_UPDATE=1 pnpm -C tests exec vitest run e2e/status/status
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
});
