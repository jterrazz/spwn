import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Colony (one world, two agents) — roster inlining + teardown semantics
 * under docker-aware mode. The colony overlay layers a second agent
 * (morpheus) and a [morpheus, neo] roster onto docker-pilot's neo. Every
 * result binds with `await using` so the spawned container is force-removed
 * at scope exit (rule B5).
 */
describe('colony', () => {
    test('roster.md inside the container names both colony members', async () => {
        // Given - a colony of neo + morpheus brought online
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .fixture('colony/')
            .exec('up');

        // Then - the container is running and neo's CLAUDE.md inlines both members
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Created container');
        const neo = result.container('neo');
        expect(neo.running).toBe(true);
        const claude = neo.file('/agents/neo/CLAUDE.md').content;
        expect(claude).toContain('## Roster');
        expect(claude).toContain('neo');
        expect(claude).toContain('morpheus');
    });

    test('destroying the colony removes the container but both agent minds survive', async () => {
        // Given - a colony brought up and then torn down in one chain
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .fixture('colony/')
            .exec(['up', 'down']);

        // Then - the container is gone but both on-disk minds survive teardown
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Destroyed');
        expect(result.stderr).toContain('project world(s) destroyed');
        expect(result.container('neo').exists).toBe(false);
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.file('spwn/agents/morpheus/SOUL.md').exists).toBe(true);
    });
});
