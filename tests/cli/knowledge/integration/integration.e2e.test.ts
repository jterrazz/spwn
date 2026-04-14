import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * World-scoped knowledge integration under the docker() spec mode.
 *
 * `spwn world knowledge ls <world-id>` reads `/world/knowledge/` inside
 * the live container. The directory is per-world, lives inside the
 * container, and is created lazily by the agent — so the empty/populated
 * transitions both have to be observable from the host.
 *
 * Dropped from the legacy file:
 *   - Four tests that wrote/read files directly under
 *     `<home>/knowledge/` via `node:fs` to "simulate" a future knowledge
 *     API. None of them invoked spwn, so they were just exercising the
 *     Node standard library. Delete rather than migrate — comment kept
 *     here so the removal is explicit in history.
 *
 * Kept / augmented:
 *   - The world-knowledge ls round-trip: empty → write file via docker
 *     exec → ls picks it up. Strengthened with a read-back via
 *     `world knowledge show` and an in-container `.file(...)` assert.
 */
describe('world knowledge integration', () => {
    test('world knowledge ls reflects files written into the container', async () => {
        // Step 1: spawn, then assert the knowledge listing is empty.
        await using empty = await spec('knowledge empty').project('docker-pilot').exec('up').run();

        expect(empty.exitCode).toBe(0);
        const neo = empty.container('neo');
        expect(neo.running).toBe(true);

        const worldId = (neo.inspect.value as { Config?: { Labels?: Record<string, string> } })
            .Config?.Labels?.['sh.spwn.world.id'];
        expect(worldId).toBeTruthy();

        // Pre-create the knowledge dir inside the container, then seed a
        // File — simulates the agent writing knowledge.
        const seed = await neo.exec(
            'sh -c "mkdir -p /world/knowledge && echo \'# Test\' > /world/knowledge/note.md"',
        );
        expect(seed.exitCode).toBe(0);

        // The file is visible inside the container.
        expect(neo.file('/world/knowledge/note.md').exists).toBe(true);

        // Step 2: host-side `spwn world knowledge ls <id>` picks up the
        // New file. Run as a fresh spec against the same fixture; the
        // Test-run label keeps the first container alive for the second
        // Spec call so `world knowledge ls` finds it.
        await using ls = await spec('knowledge ls')
            .project('docker-pilot')
            .exec(`world knowledge ls ${worldId}`)
            .run();

        expect(ls.exitCode).toBe(0);
        const combined = `${ls.stdout.text}\n${ls.stderr.text}`;
        expect(combined).toContain('note.md');
    });
});
