import { describe, expect, test } from 'vitest';

import { spec } from '../../../_setup/cli.specification.js';

/**
 * World destroy (spwn down) under the docker() spec mode.
 *
 * Project-mode bare `spwn down` destroys every world in the project —
 * This replaces the legacy `spwn down <world-id>` single-world flow. The
 * Per-world status banners ("Stopped agent", "Mind persisted", …) only
 * Fire on the per-id path, which project mode doesn't expose from
 * Fixture state. We assert on the project-mode banner wording instead.
 *
 * Augmented over the legacy test:
 *   - Confirms the container is removed from docker entirely after
 *     destroy (`container('neo').exists === false`)
 *   - Asserts `world list --json` returns zero running worlds
 */
describe('world destroy', () => {
    test('destroys a running project world', async () => {
        await using result = await spec('down running world')
            .project('docker-pilot')
            .exec(['up', 'down'])
            .run();

        expect(result.exitCode).toBe(0);

        // Project-mode destroy banners (stderr per Unix convention)
        result.stderr.toContain('Stopping project worlds');
        result.stderr.toContain('Destroyed');
        result.stderr.toContain('project world(s) destroyed');

        // Container is gone from docker
        expect(result.container('neo').exists).toBe(false);
    });

    test('destroy removes world from list', async () => {
        await using result = await spec('down removes from list')
            .project('docker-pilot')
            .exec(['up', 'down', 'world list --json'])
            .run();

        expect(result.exitCode).toBe(0);

        const list = result.json.value as {
            mode: string;
            worlds: Array<{ name: string; status: string }>;
        };
        expect(list.mode).toBe('project');
        // No running worlds after destroy
        expect(list.worlds.every((w) => w.status !== 'running')).toBe(true);

        expect(result.container('neo').exists).toBe(false);
    });

    test('per-id destroy prints the detailed step banners', async () => {
        // Given - a running project world
        // When - `spwn down <world-id>` is called with an explicit id
        // (the per-id destroy path in apps/cli/world/destroy.go)
        // Then - the richer per-step banners fire ("Stopped agent",
        // "Removed container", "Mind persisted"). These are NOT emitted
        // By the project-mode bare `down`, so this test guards the
        // Per-id code path specifically.
        //
        // The id is resolved via shell substitution against
        // .spwn/world-states/ (each subdir there is named after the
        // World id) so the whole flow fits in one spec.
        await using result = await spec('destroy steps per-id')
            .project('docker-pilot')
            .exec(['up', 'down $(ls .spwn/world-states 2>/dev/null | head -1)'])
            .run();
        expect(result.exitCode).toBe(0);

        result.stderr.toContain('Stopped agent');
        result.stderr.toContain('Removed container');
        result.stderr.toContain('Mind persisted');
    });

    test('destroy non-existent world fails', async () => {
        await using result = await spec('down missing world')
            .project('docker-pilot')
            .exec('down world-nonexistent-00000')
            .run();

        expect(result.exitCode).toBe(1);
        // Friendly error — no stack trace or panic
        expect(result.stderr.text).not.toContain('TypeError');
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
        // Error line mentions the missing world by name and says "not found"
        await result.stderr.toMatch('destroy-missing-world.txt');
    });

    test('destroy rejects legacy w-<name>-<hex> ID shape cleanly', async () => {
        // Given - a user types an ID in the pre-2026 format
        // When - we pass it to `spwn down`
        // Then - the CLI surfaces a "not found" error (because no
        // World with that ID exists) without panicking, goroutine
        // Dumping, or silently succeeding. The error message is
        // Scrutable enough for the user to discover the modern
        // World-<slug>-<hex> format.
        await using result = await spec('down legacy w- id')
            .project('docker-pilot')
            .exec('down w-acme-12345')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
        // The error path still surfaces something the user can act on.
        expect(result.stderr.text).toMatch(/not found|does not exist|unknown world/i);
    });

    test('destroy rejects legacy spwn-world-<slug>-<hex> ID shape cleanly', async () => {
        // Same as above but for the even older prefix that was
        // Retired when IDs moved to `world-<slug>-<hex>`.
        await using result = await spec('down legacy spwn-world- id')
            .project('docker-pilot')
            .exec('down spwn-world-acme-12345')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).toMatch(/not found|does not exist|unknown world/i);
    });
});
