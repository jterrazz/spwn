import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * World destroy (`spwn down`) under docker-aware mode. Project-mode bare
 * `spwn down` destroys every world in the project; the richer per-id
 * banners ("Stopped agent", "Removed container", "Mind persisted") only
 * fire on the explicit `spwn down <id>` path. Every spec here brings a
 * world up and reads the container back, which is why they stay chains;
 * each result binds with `await using` (rule B5). The two refusals that
 * spawn nothing — an undeclared world key, a pre-2026 id — are documents
 * beside this file.
 */
describe('world destroy', () => {
    test('destroys a running project world', async () => {
        // Given - a running project world brought up then torn down
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec(['up', 'down']);

        // Then - the project-mode destroy banners fire and the container is gone
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Stopping project worlds');
        expect(result.stderr).toContain('Destroyed');
        expect(result.stderr).toContain('project world(s) destroyed');
        expect(result.container('neo').exists).toBe(false);
    });

    test('destroy removes world from list', async () => {
        // Given - a world upped, destroyed, then listed as JSON
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'down', 'world list --json']);

        // Then - no running worlds remain and the container is gone
        expect(result.exitCode).toBe(0);
        const list = result.json.value as {
            mode: string;
            worlds: { name: string; status: string }[];
        };
        expect(list.mode).toBe('project');
        expect(list.worlds.every((world) => world.status !== 'running')).toBe(true);
        expect(result.container('neo').exists).toBe(false);
    });

    test('per-id destroy prints the detailed step banners', async () => {
        // Given - a running world torn down by its explicit id (resolved from .spwn/world-states)
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'down $(ls .spwn/world-states 2>/dev/null | head -1)']);

        // Then - the per-id path emits the richer per-step banners
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Stopped agent');
        expect(result.stderr).toContain('Removed container');
        expect(result.stderr).toContain('Mind persisted');
    });
});
