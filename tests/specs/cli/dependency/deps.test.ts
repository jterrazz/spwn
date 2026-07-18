import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn check` dependencies-field resolution. Mirrors the tools-field goldens
 * in ../check/ against the frozen check-valid-dep / check-invalid-dep fixtures,
 * whose neo agent declares refs via the `dependencies:` field. The runner is
 * docker-aware, so every result binds with `await using` even though check
 * spawns no containers (rule B5).
 */

test('accepts an spwn:mempalace dependency without error', async () => {
    // Given - check-valid-dep declares dependencies: ["spwn:mempalace"]
    await using result = await cli.fixture('$FIXTURES/check-valid-dep/').exec('check');

    // Then - check passes with the canonical success report
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toMatch('valid-dep.txt');
});

test('rejects a nonexistent dependency ref with the same wording as tools', async () => {
    // Given - check-invalid-dep declares a bogus dependency ref
    await using result = await cli.fixture('$FIXTURES/check-invalid-dep/').exec('check');

    // Then - check fails with the deterministic violation report naming the dep
    expect(result.exitCode).toBe(1);
    expect(result.stdout).toMatch('invalid-dep.txt');
});
