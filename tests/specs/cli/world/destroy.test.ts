import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * World destroy (`spwn down`) under docker-aware mode. Project-mode bare
 * `spwn down` destroys every world in the project; the richer per-id
 * banners ("Stopped agent", "Removed container", "Mind persisted") only
 * fire on the explicit `spwn down <id>` path. Error cases are pinned as
 * full stderr goldens where the wording is stable, and probed with a
 * regex where the id is arbitrary. Every result binds with `await using`
 * (rule B5).
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
            worlds: Array<{ name: string; status: string }>;
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

    test('destroy non-existent world fails cleanly', async () => {
        // Given - a down against a world key that does not exist
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('down world-nonexistent-00000');

        // Then - exits non-zero with the canonical "not found" banner (full stderr golden)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('destroy-missing-world.txt');
    });

    test('destroy rejects legacy id shapes cleanly', async () => {
        // Given - a user types an id in a pre-2026 format
        for (const legacyId of ['w-acme-12345', 'spwn-world-acme-12345']) {
            await using result = await cli
                .fixture('$FIXTURES/docker-pilot/')
                .exec(`down ${legacyId}`);

            // Then - non-zero with an actionable "not found" message, no crash (scalpel: arbitrary id, regex + absence probes)
            expect(result.exitCode).not.toBe(0);
            expect(result.stderr).not.toContain('panic');
            expect(result.stderr).not.toContain('goroutine');
            expect(result.stderr.text).toMatch(/not found|does not exist|unknown world/i);
        }
    });
});
