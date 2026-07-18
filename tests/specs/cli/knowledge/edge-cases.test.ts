import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Edge cases for world-scoped knowledge, exercised through `spwn check`.
 * Knowledge is opt-in per-world via `worlds.<name>.knowledge` in spwn.yaml.
 * These cover the fringes — a declared path missing on disk, a project mixing
 * a knowledge world with one without, and a path escaping the project root.
 * The check reports are deterministic, so each is a full golden with the temp
 * cwd tokenised. Every result binds with `await using` (rule B5); no docker.
 */

test('missing knowledge directory surfaces a warning, not a crash', async () => {
    // Given - knowledge-missing-dir declares worlds.neo.knowledge but ships no such dir
    await using result = await cli.fixture('$FIXTURES/knowledge-missing-dir/').exec('check');

    // Then - check stays green and the report warns with a concrete mkdir fix
    expect(result.exitCode, result.stdout.text).toBe(0);
    expect(result.stdout).toMatch('knowledge-missing-dir.txt');
});

test('mixed project (one world with knowledge, one without) passes check', async () => {
    // Given - knowledge-mixed-worlds: primary opts into knowledge, ephemeral does not
    await using result = await cli.fixture('$FIXTURES/knowledge-mixed-worlds/').exec('check');

    // Then - check passes; the info hint fires only for the world that didn't opt in
    expect(result.exitCode, result.stdout.text).toBe(0);
    expect(result.stdout).toMatch('knowledge-mixed-worlds.txt');
});

test('knowledge path outside project root is a warning today, not an error', async () => {
    // Given - knowledge-outside-root declares knowledge: ../escape (outside the project)
    await using result = await cli.fixture('$FIXTURES/knowledge-outside-root/').exec('check');

    // Then - today this is a permissive warning, not a rejection (golden pins current behaviour)
    expect(result.exitCode, result.stdout.text).toBe(0);
    expect(result.stdout).toMatch('knowledge-outside-root.txt');
});
