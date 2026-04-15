import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Multi-agent colony (one world, multiple agents) under the docker()
 * spec mode.
 *
 * Seeds:
 *   - agent/morpheus/: a second agent mind alongside docker-pilot's neo
 *   - spwn.yaml/colony.yaml: overrides worlds.neo to list [morpheus, neo]
 *
 * The seed handler for `spwn.yaml/` shallow-merges top-level keys, so
 * redeclaring `worlds.neo` replaces the whole neo entry with the colony
 * roster.
 *
 * Legacy semantics preserved:
 *   - up succeeds with banners for container + agent alive
 *   - `Colony spawned <N> agent(s)` (or equivalent structured report)
 *   - both agents' minds are on disk in spwn/agents/<name>
 *   - destroy (down) tears down the world cleanly
 *
 * Augmented over the legacy test:
 *   - Asserts both agents' homes are docker-cp'd into the container
 *     at /agents/morpheus and /agents/neo at spawn time
 *   - Reads `agent ls --json` and `world list --json` to confirm both
 *     agents are attached to the same running world
 */
describe('colony multi-agent', () => {
    test('up spawns one world containing two agents', async () => {
        await using result = await spec('colony up')
            .project('docker-pilot')
            .seed('agent/morpheus')
            .seed('spwn.yaml/colony.yaml')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Created container');
        result.stderr.toContain('Agent is alive');

        // Both agents' on-disk minds are present in the project tree.
        expect(result.file('spwn/agents/neo/identity/profile.md').exists).toBe(true);
        expect(result.file('spwn/agents/morpheus/identity/profile.md').exists).toBe(true);

        // And docker-cp'd into the world container at /agents/<name>.
        const neo = result.container('neo');
        expect(neo.running).toBe(true);
        expect(neo.file('/agents/neo').exists).toBe(true);
        expect(neo.file('/agents/morpheus').exists).toBe(true);
        expect(neo.file('/agents/neo/identity/profile.md').exists).toBe(true);
        expect(neo.file('/agents/morpheus/identity/profile.md').exists).toBe(true);

        const ls = await neo.exec('ls /agents');
        expect(ls.exitCode).toBe(0);
        ls.stdout.toContain('neo');
        ls.stdout.toContain('morpheus');
    });

    test('agent ls --json and world list --json both show the full colony', async () => {
        await using result = await spec('colony ls')
            .project('docker-pilot')
            .seed('agent/morpheus')
            .seed('spwn.yaml/colony.yaml')
            .exec(['up', 'world list --json'])
            .run();

        expect(result.exitCode).toBe(0);

        const list = result.json.value as {
            mode: string;
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(list.mode).toBe('project');
        expect(list.worlds).toHaveLength(1);
        expect(list.worlds[0].name).toBe('neo');
        expect(list.worlds[0].status).toBe('running');
        expect(list.worlds[0].agents).toEqual(expect.arrayContaining(['neo', 'morpheus']));
        expect(list.worlds[0].agents).toHaveLength(2);

        // And the container is still live
        expect(result.container('neo').running).toBe(true);
    });

    test('destroying the colony cleans up both agents at once', async () => {
        await using result = await spec('colony down')
            .project('docker-pilot')
            .seed('agent/morpheus')
            .seed('spwn.yaml/colony.yaml')
            .exec(['up', 'down'])
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Destroyed');
        result.stderr.toContain('project world(s) destroyed');

        // Container gone from docker entirely
        expect(result.container('neo').exists).toBe(false);
    });
});
