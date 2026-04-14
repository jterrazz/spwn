import { describe, expect, test } from 'vitest';

import { dockerSpec } from '../../../setup/cli.specification.js';

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
        await using result = await dockerSpec('down running world')
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
        await using result = await dockerSpec('down removes from list')
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

    test('destroy non-existent world fails', async () => {
        await using result = await dockerSpec('down missing world')
            .project('docker-pilot')
            .exec('down w-nonexistent-00000')
            .run();

        expect(result.exitCode).not.toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        // Friendly error — no stack trace or panic
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');
        // Error line mentions the missing world by name
        expect(combined).toContain('w-nonexistent-00000');
        expect(combined).toContain('not found');
    });
});
