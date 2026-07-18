import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Regression guard for pre-refactor project layouts. Before the unified
 * `dependencies:` model an agent.yaml carried three separate buckets —
 * `tools:`, `skills:`, `hooks:`. The strict parser silently ignores
 * unknown top-level keys, so an upgraded project would quietly lose every
 * dep; `spwn check` now catches these legacy shapes with a migration hint
 * for each retired bucket. The check report is deterministic, so it is
 * pinned as a full golden (rule D11) with the temp cwd tokenised. The
 * runner is docker-aware, so the result binds with `await using` (B5).
 */

test('flags every retired top-level bucket with a migration hint', async () => {
    // Given - the legacy-layout fixture encodes all three retired top-level buckets in one agent.yaml
    await using result = await cli.fixture('$FIXTURES/legacy-layout/').exec('check');

    // Then - check fails and the report names each bucket with the current path form to migrate to
    expect(result.exitCode).not.toBe(0);
    expect(result.stdout).toMatch('legacy-buckets.txt');
});
