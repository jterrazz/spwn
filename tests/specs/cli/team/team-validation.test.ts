import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Input validation for `spwn team new`. The team subsystem is
 * experimental, but the CLI must still refuse obviously-broken names so a
 * downstream slug fallback can't produce a team literally named "team"
 * (QA finding #28). The refusal message is deterministic, so it is pinned
 * as a full golden (rule D11). Each spec gets a fresh empty project and an
 * isolated SPWN_HOME; the runner is docker-aware, so every result binds
 * with `await using` (rule B5).
 */

const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('spwn team new validation', () => {
    test('rejects an empty name', async () => {
        // Given - an isolated home and an empty-string team name
        await using result = await isolated().exec('team new ""');

        // Then - exits non-zero with the canonical "team name is required" error
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('team-required.txt');
    });

    test('rejects a whitespace-only name', async () => {
        // Given - an isolated home and a spaces-only team name
        await using result = await isolated().exec('team new "   "');

        // Then - exits non-zero with the same "required" error
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('team-required.txt');
    });
});
