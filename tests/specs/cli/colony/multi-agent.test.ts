import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Multi-agent colony (one world, multiple agents) under docker-aware mode.
 * The colony overlay adds morpheus alongside docker-pilot's neo and rewrites
 * worlds.neo to the [morpheus, neo] roster. Asserts both agent homes are
 * docker-cp'd into the world container and that both list views agree.
 */
describe('colony multi-agent', () => {
    test('up spawns one world containing two agents', async () => {
        // Given - a two-agent colony brought online
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .fixture('colony/')
            .exec('up');

        // Then - both minds are on disk and docker-cp'd into the running container
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Created container');
        expect(result.stderr).toContain('Agent is alive');
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.file('spwn/agents/morpheus/SOUL.md').exists).toBe(true);
        const neo = result.container('neo');
        expect(neo.running).toBe(true);
        expect(neo.file('/agents/neo').exists).toBe(true);
        expect(neo.file('/agents/morpheus').exists).toBe(true);
        expect(neo.file('/agents/neo/SOUL.md').exists).toBe(true);
        expect(neo.file('/agents/morpheus/SOUL.md').exists).toBe(true);
        const ls = await neo.exec('ls /agents');
        expect(ls.exitCode).toBe(0);
        expect(ls.stdout).toContain('neo');
        expect(ls.stdout).toContain('morpheus');
    });

    test('world list --json shows the full colony', async () => {
        // Given - the colony brought up, then listed as JSON
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .fixture('colony/')
            .exec(['up', 'world list --json']);

        // Then - one running project world attaches both agents
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
        expect(result.container('neo').running).toBe(true);
    });

    test('destroying the colony cleans up both agents at once', async () => {
        // Given - the colony brought up then torn down in one chain
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .fixture('colony/')
            .exec(['up', 'down']);

        // Then - the container is fully gone from docker
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Destroyed');
        expect(result.stderr).toContain('project world(s) destroyed');
        expect(result.container('neo').exists).toBe(false);
    });
});
