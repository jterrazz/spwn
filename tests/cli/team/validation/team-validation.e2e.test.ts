import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Input validation for `spwn team new`.
 *
 * The team subsystem is experimental, but the CLI still needs to refuse
 * obviously-broken inputs so downstream slug fallbacks can't produce a
 * team literally named "team" (QA finding #28).
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('spwn team new validation', () => {
    test('rejects an empty name', async () => {
        // Given - an isolated home
        // When - running team new with an empty string
        // Then - exit 1, "team name is required" on stderr
        const result = await isolated('team new empty').exec('team new ""').run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toMatch(/team name is required/i);
    });

    test('rejects a whitespace-only name', async () => {
        // Given - an isolated home
        // When - running team new with only spaces
        // Then - exit 1, same "required" error
        const result = await isolated('team new blank').exec('team new "   "').run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toMatch(/team name is required/i);
    });
});
