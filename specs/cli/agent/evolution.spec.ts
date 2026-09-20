import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * The one evolution spec whose subject is a line that must NOT be in the
 * written reflexion — and whose file may legitimately not be written at
 * all. `files:` says what a file contains, never what it does not, and
 * has no form for "if it exists"; so this one stays a chain while dream,
 * sleep and fork are documents beside it.
 *
 * Mtime-sensitive behaviours (stale playbook archival, session pruning)
 * are not covered here — fixture layering does not backdate files, so
 * those assertions belong to the Go unit tests. The runner is
 * docker-aware, so the result binds with `await using` (rule B5).
 */

test('dream writes no empty-id session row', async () => {
    // Given - a fresh agent dreamt with no real journal entries
    await using result = await cli.fixture('$FIXTURES/empty/').exec(['init', 'agent dream neo']);

    // Then - if a reflexion is written, no row carries an empty session id
    expect(result.exitCode).toBe(0);
    const reflexion = result.file('spwn/agents/neo/playbooks/auto-reflexion.md');
    if (reflexion.exists) {
        expect(reflexion.content).not.toMatch(/^- :/m);
    }
});
