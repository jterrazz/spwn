import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Regression suite for pre-refactor project layouts.
 *
 * Before the unified `dependencies:` model, an agent.yaml carried
 * three separate buckets: `tools:`, `skills:`, `hooks:`. The strict-
 * typed parser silently ignores unknown top-level keys, so an
 * upgraded project would quietly lose every dep. `spwn check` now
 * catches these legacy shapes with a migration hint so the user's
 * first post-upgrade command tells them exactly what to change.
 *
 * This file is the guardrail: if that rule is ever removed or
 * downgraded, these assertions fail loudly.
 */

describe('legacy agent.yaml shapes', () => {
    test('top-level tools/skills/hooks blocks are errors with migration hints', async () => {
        // Given - the legacy-layout fixture encodes all three
        // Retired top-level buckets in one agent.yaml.
        // When - `spwn check` runs
        // Then - each bucket produces a distinct error naming the
        // Key, with a hint pointing at the current flat
        // `dependencies:` list. Exit code is non-zero so scripts
        // Notice the regression surface.
        const result = await spec('legacy top-level buckets')
            .project('legacy-layout')
            .exec('check')
            .run();

        expect(result.exitCode).not.toBe(0);

        const report = result.stdout.text;
        expect(report).toContain('legacy top-level "tools" block');
        expect(report).toContain('legacy top-level "skills" block');
        expect(report).toContain('legacy top-level "hooks" block');

        // Every hint must point at the flat dependencies: list so
        // The user knows where the entries should move to.
        expect(report).toMatch(/dependencies:/);
    });

    test('migration hints include the scheme form for each retired bucket', async () => {
        // Each legacy bucket maps to an explicit scheme prefix:
        //   tools:  → tool:<name>
        //   skills: → skill:<name>
        //   hooks:  → hook:<name>
        // The hint should show the right prefix for each so the
        // User can copy-paste from the hint into their new list.
        const result = await spec('legacy hints show schemes')
            .project('legacy-layout')
            .exec('check')
            .run();

        expect(result.exitCode).not.toBe(0);
        const report = result.stdout.text;
        expect(report).toMatch(/tool:<name>/);
        expect(report).toMatch(/skill:<name>/);
        expect(report).toMatch(/hook:<name>/);
    });
});
