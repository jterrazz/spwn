import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn world list --json` coverage. The text path is exercised via the
 * lifecycle/spawn snapshots; this file pins the JSON envelope so the
 * declared-world merge (name, status, agents) stays structurally stable
 * independent of human formatting. CLI-only, but the runner is
 * docker-aware so every result binds with `await using` (rule B5).
 */

test('world ls is an alias for world list', async () => {
    // Given - single-agent declares one world neo, listed via both spellings
    await using viaLs = await cli.fixture('$FIXTURES/single-agent/').exec('world ls --json');
    await using viaList = await cli.fixture('$FIXTURES/single-agent/').exec('world list --json');

    // Then - the short form emits byte-identical JSON to the canonical form
    expect(viaLs.exitCode).toBe(0);
    expect(viaList.exitCode).toBe(0);
    expect(viaLs.stdout.text).toBe(viaList.stdout.text);
});

test('reports declared worlds for a project', async () => {
    // Given - single-agent declares one world neo with one agent
    await using result = await cli.fixture('$FIXTURES/single-agent/').exec('world list --json');

    // Then - the JSON envelope lists the declared world in stopped state
    expect(result.exitCode).toBe(0);
    expect(result.json).toMatch('project.json');
});
