import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * World-scoped knowledge integration under docker-aware mode.
 * `spwn world knowledge ls <world-id>` reads `/world/knowledge/` inside the
 * live container; the directory is per-world and created lazily, so the
 * empty/populated transition is observed from the host. Every result binds
 * with `await using` so the container is force-removed at scope exit (rule B5).
 *
 * Dropped from the legacy file: four tests that wrote/read files under a home
 * dir via node:fs without ever invoking spwn — they exercised the Node stdlib,
 * not the product, so they are deleted rather than migrated.
 */
describe('world knowledge integration', () => {
    test('world knowledge ls reflects files written into the container', async () => {
        // Given - a docker-pilot world brought online
        await using empty = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

        expect(empty.exitCode).toBe(0);
        const neo = empty.container('neo');
        expect(neo.running).toBe(true);

        const worldId = (neo.inspect.value as { Config?: { Labels?: Record<string, string> } })
            .Config?.Labels?.['sh.spwn.world.id'];
        expect(worldId).toBeTruthy();

        // When - the knowledge dir is seeded inside the container (simulates the agent writing)
        const seed = await neo.exec(
            'sh -c "mkdir -p /world/knowledge && echo \'# Test\' > /world/knowledge/note.md"',
        );
        expect(seed.exitCode).toBe(0);
        expect(neo.file('/world/knowledge/note.md').exists).toBe(true);

        // Then - host-side `world knowledge ls` picks up the new file (container-log probe)
        await using ls = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(`world knowledge ls ${worldId}`);

        expect(ls.exitCode).toBe(0);
        expect(ls.stdout).toContain('note.md');
    });
});
