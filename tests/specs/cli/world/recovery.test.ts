import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Error recovery under docker-aware mode. Covers the friendly-error
 * surface spwn exposes for common mistakes: double destroy, inspecting a
 * ghost world, spawning a bad config (no container leaked), a failing
 * command not corrupting later state, and a full up→down→up recreate.
 * Each operation binds its own `await using` (rule B5); the first run's
 * scope still owns cleanup of any container it spawned.
 */
describe('error recovery', () => {
    test('double destroy — second destroy fails with a clean not-found error', async () => {
        // Given - a world upped then destroyed, so the container is already gone
        await using up = await cli.fixture('$FIXTURES/docker-pilot/').exec(['up', 'down']);
        expect(up.exitCode).toBe(0);
        expect(up.container('neo').exists).toBe(false);

        // When - a second down runs against the clean project
        await using secondDown = await cli.fixture('$FIXTURES/docker-pilot/').exec('down neo');

        // Then - non-zero with a clean message, no stack trace or usage dump (scalpel: dynamic error wording + absence probes)
        expect(secondDown.exitCode).toBe(1);
        expect(secondDown.stderr.text).toMatch(/not (?:found|running)|no (?:running )?worlds?/i);
        expect(secondDown.stderr).not.toContain('panic');
        expect(secondDown.stderr).not.toContain('goroutine');
        expect(secondDown.stdout).not.toContain('Available Commands:');
        expect(secondDown.stdout).not.toContain('Global Flags:');
    });

    test('world inspect on a non-existent id fails cleanly', async () => {
        // Given - a world inspect against a ghost id
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('world inspect world-ghost-99999');

        // Then - non-zero "not found" with no usage dump or crash (scalpel: dynamic error wording + absence probes)
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toMatch(/not found/i);
        expect(result.stdout).not.toContain('Available Commands:');
        expect(result.stderr).not.toContain('panic');
        expect(result.stderr).not.toContain('goroutine');
    });

    test('spawn with an invalid world name leaks no container', async () => {
        // Given - a project-mode up requesting a world key that is not in the manifest
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('up nonexistent-world');

        // Then - non-zero and no container left behind (scalpel: absence probes)
        expect(result.exitCode).toBe(1);
        expect(result.container('neo').exists).toBe(false);
        expect(result.container('nonexistent-world').exists).toBe(false);
        expect(result.stderr).not.toContain('panic');
        expect(result.stderr).not.toContain('goroutine');
    });

    test('an error does not corrupt state — the next command still works', async () => {
        // Given - a failing destroy, then a normal list in a fresh project dir
        await using errorResult = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('down world-ghost-00000');
        expect(errorResult.exitCode).toBe(1);

        await using listResult = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('world list --json');

        // Then - the follow-up command succeeds and reports project mode
        expect(listResult.exitCode).toBe(0);
        const list = listResult.json.value as {
            mode: string;
            worlds: Array<{ name: string; status: string }>;
        };
        expect(list.mode).toBe('project');
    });

    test('spawn and destroy cycle — world can be recreated after destroy', async () => {
        // Given - up, down, up again in one chain
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'down', 'up']);

        // Then - the final state is a running neo container
        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
    });
});
