import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Smallest docker-aware coverage — the reference for the end-to-end flow:
 * the runner injects SPWN_TEST_LABEL into the child env, spwn stamps it as
 * a Docker label on every container it spawns, and `await using` force-
 * removes every tracked container when the scope exits (rule B5) so
 * parallel runs never collide and nothing leaks.
 */
describe('world up/down (docker pilot)', () => {
    test('spwn up brings a declared world into running state', async () => {
        // Given - the docker-pilot fixture with one declared world neo
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

        // Then - the neo container exists and is running
        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
        expect(neo.status).toBe('running');
    });

    test('spwn up on a running world is idempotent', async () => {
        // Given - a world brought up twice then listed as JSON
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec(['up', 'up', 'world list --json']);

        // Then - exactly one running neo world (no duplicate)
        expect(result.exitCode).toBe(0);
        const list = result.json.value as {
            worlds: Array<{ name: string; status: string }>;
        };
        const neoWorlds = list.worlds.filter((world) => world.name === 'neo');
        expect(neoWorlds).toHaveLength(1);
        expect(neoWorlds[0].status).toBe('running');
    });

    test('spwn up is repeatable without host-side cleanup', async () => {
        // Given - up, down, up again with no manual cleanup between steps
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec(['up', 'down', 'up']);

        // Then - exit zero and no stale .codex artefact left on the host
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo/.codex').exists).toBe(false);
    });

    test('up then down removes the container entirely', async () => {
        // Given - up followed by down (spwn down fully destroys, not just stops)
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec(['up', 'down']);

        // Then - the post-run label lookup finds no container
        expect(result.exitCode).toBe(0);
        expect(result.container('neo').exists).toBe(false);
    });
});
