import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Error recovery under the docker() spec mode.
 *
 * Covers the "friendly error" surface spwn exposes for common mistakes:
 *   - Double destroy (second destroy must surface a clean "not found")
 *   - Inspect non-existent world
 *   - Spawn with a bad config (no container leaked)
 *   - A failing command must not corrupt state for the next call
 *   - Up → down → up again should succeed end-to-end
 *
 * Two operations require the live container id (not the world key):
 * spwn world inspect and spwn down <id>. We capture the id from the
 * first run via `container('neo').id` and feed it into a fresh
 * spec call. The first run's `await using` still owns cleanup
 * of any container it spawned.
 */
describe('error recovery', () => {
    test('double destroy — second destroy fails with a clean not-found error', async () => {
        // Up, capture the container id, then destroy via project-mode down.
        await using up = await spec('double destroy up')
            .project('docker-pilot')
            .exec(['up', 'down'])
            .run();

        expect(up.exitCode).toBe(0);
        // The container is gone; a second down must fail gracefully.
        expect(up.container('neo').exists).toBe(false);

        // A second `spwn down` on a clean project must exit non-zero with
        // A clean message (the neo world has no running container).
        await using secondDown = await spec('double destroy second')
            .project('docker-pilot')
            .exec('down neo')
            .run();

        expect(secondDown.exitCode).not.toBe(0);
        expect(secondDown.stderr.text).toMatch(/not (found|running)|no (running )?worlds?/i);
        // No stack trace, no usage dump.
        expect(secondDown.stderr.text).not.toContain('TypeError');
        expect(secondDown.stderr.text).not.toContain('panic');
        expect(secondDown.stderr.text).not.toContain('goroutine');
        expect(secondDown.stdout.text).not.toContain('Available Commands:');
        expect(secondDown.stdout.text).not.toContain('Global Flags:');
    });

    test('world inspect on a non-existent id fails cleanly', async () => {
        await using result = await spec('inspect missing')
            .project('docker-pilot')
            .exec('world inspect w-ghost-99999')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toMatch(/not found/i);
        expect(result.stdout.text).not.toContain('Available Commands:');
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
    });

    test('spawn with an invalid project name leaks no container', async () => {
        // Project-mode up: request a world key that doesn't exist in the
        // Docker-pilot manifest.
        await using result = await spec('spawn bad project world')
            .project('docker-pilot')
            .exec('up nonexistent-world')
            .run();

        expect(result.exitCode).not.toBe(0);

        // And no container was left behind.
        expect(result.container('neo').exists).toBe(false);
        expect(result.container('nonexistent-world').exists).toBe(false);

        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
    });

    test('an error does not corrupt state — the next command still works', async () => {
        // First: trigger an error (destroy a non-existent world). The
        // Array chain short-circuits on failure, so we split across two
        // DockerSpec calls using the same project fixture.
        await using errorResult = await spec('error first')
            .project('docker-pilot')
            .exec('down w-ghost-00000')
            .run();
        expect(errorResult.exitCode).not.toBe(0);

        // Then: a normal command in a fresh project dir still works.
        await using listResult = await spec('list after error')
            .project('docker-pilot')
            .exec('world list --json')
            .run();
        expect(listResult.exitCode).toBe(0);

        const list = listResult.json.value as {
            mode: string;
            worlds: Array<{ name: string; status: string }>;
        };
        expect(list.mode).toBe('project');
    });

    test('spawn and destroy cycle — world can be recreated after destroy', async () => {
        // Up → down → up again, all in one run.
        await using result = await spec('spawn destroy recreate')
            .project('docker-pilot')
            .exec(['up', 'down', 'up'])
            .run();

        expect(result.exitCode).toBe(0);
        // The final state is a running neo container.
        const neo = result.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
    });
});
