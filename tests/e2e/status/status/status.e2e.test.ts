import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn status` — top-level dashboard.
 *
 * Status output is written to stderr, which the `@jterrazz/test`
 * ExecAdapter discards on exit 0. That means we cannot snapshot the
 * happy-path output from this runner. The legacy file's content
 * assertions ("Architect offline", "Worlds", "512m", etc.) are
 * therefore weakened to exit-code smoke tests here; the richer
 * coverage lives in the Docker-backed status test that still uses
 * the legacy helpers (kept under status-docker.e2e.test.ts).
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('spwn status', () => {
    describe('without worlds', () => {
        test('runs cleanly after init', async () => {
            const init = await isolated('init').exec('init').run();
            expect(init.exitCode).toBe(0);

            const result = await isolated('status').exec('status').run();
            expect(result.exitCode).toBe(0);
        });
    });

    describe('error handling', () => {
        test('status on uninitialised home still works', async () => {
            const result = await isolated('status no init').exec('status').run();
            expect(result.exitCode).toBe(0);
        });
    });
});
