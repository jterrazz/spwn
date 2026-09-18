import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn upgrade --check` queries the GitHub release feed, so both the tag it
 * reports and whether it reached the network at all vary run to run. That is
 * the one shape a byte-exact `stdout:` cannot carry, so this stays a probe on
 * the two lines that never move. The result binds with `await using`
 * (rule B5); no docker.
 */

describe('cli upgrade', () => {
    test('upgrade --check queries the release feed and prints a version', async () => {
        // Given - the upgrade command hits the GitHub release feed
        await using result = await cli.fixture('$FIXTURES/empty/').exec('upgrade --check');

        // Then - the stable banner lines render (scalpel: latest tag + network state are dynamic)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Current version');
        expect(result.stderr).toContain('Checking for updates');
    });
});
