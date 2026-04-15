import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Smallest docker() spec mode coverage — the reference for the
 * end-to-end flow:
 *
 *   1. spec injects SPWN_TEST_LABEL=<id> into the child env
 *   2. spwn stamps that id as a Docker label on every container it spawns
 *   3. After the command returns, the framework queries containers by
 *      that label and exposes them via `result.container(name)`
 *   4. `await using` force-removes every tracked container when the
 *      scope exits, so tests running in parallel never collide and
 *      nothing leaks between runs
 */
describe('world up/down (docker pilot)', () => {
    test('spwn up brings a declared world into running state', async () => {
        await using result = await spec('up docker-pilot').project('docker-pilot').exec('up').run();

        expect(result.exitCode).toBe(0);

        const neo = result.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
        expect(neo.status).toBe('running');
    });

    test('spwn up on a running world is idempotent', async () => {
        // Given - a world that's already up
        // When - up is called again for the same world
        await using result = await spec('double up')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec(['up', 'up', 'world list --json'])
            .run();

        // Then - exit zero, one world present (no duplicate)
        expect(result.exitCode).toBe(0);
        const list = result.json.value as {
            worlds: Array<{ name: string; status: string }>;
        };
        const neoWorlds = list.worlds.filter((w) => w.name === 'neo');
        expect(neoWorlds).toHaveLength(1);
        expect(neoWorlds[0].status).toBe('running');
    });

    test('spwn up is repeatable without host-side cleanup', async () => {
        // Given - up followed by down followed by up again
        // When - the whole chain runs with no manual cleanup between steps
        await using result = await spec('up-down-up clean')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec(['up', 'down', 'up'])
            .run();

        // Then - exit zero and no stale .codex/ artefact on the host
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo/.codex').exists).toBe(false);
    });

    test('up then down removes the container entirely', async () => {
        // Given - up followed by down (spwn down fully destroys the container,
        // It doesn't just stop it, so the framework's post-run query finds
        // No container tagged with this test's label).
        await using result = await spec('up-then-down')
            .project('docker-pilot')
            .exec(['up', 'down'])
            .run();

        expect(result.exitCode).toBe(0);

        const neo = result.container('neo');
        expect(neo.exists).toBe(false);
    });
});
