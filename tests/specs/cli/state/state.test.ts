import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * State tracking under docker-aware mode. In project mode the source of truth
 * is docker label queries surfaced via `world list --json`; each assertion is
 * cross-checked against the live container status. Every result binds with
 * `await using` so the container is force-removed at scope exit (rule B5).
 *
 * Multi-exec chains only expose the last command's streams; to assert on an
 * earlier step's banners the chain is split, keeping the earlier result in
 * scope under the test-run label so follow-up queries still find its container.
 */
describe('state tracking', () => {
    test('after up, banners fire and world list --json reports the world', async () => {
        // Given - a docker-pilot world brought online
        await using up = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

        expect(up.exitCode).toBe(0);
        expect(up.stderr).toContain('Created container');
        expect(up.stderr).toContain('Agent is alive');

        const neo = up.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);

        // When - a second run queries the list; the first container is still live under the label
        await using list = await cli.fixture('$FIXTURES/docker-pilot/').exec('world list --json');

        // Then - one running project world named neo is reported
        expect(list.exitCode).toBe(0);
        const report = list.json.value as {
            mode: string;
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(report.mode).toBe('project');
        expect(report.worlds).toHaveLength(1);
        expect(report.worlds[0]).toEqual({
            agents: ['neo'],
            name: 'neo',
            status: 'running',
        });
    });

    test('down removes the world from state and from docker', async () => {
        // Given - up then down so the last step carries the destroy banners
        await using destroy = await cli.fixture('$FIXTURES/docker-pilot/').exec(['up', 'down']);

        expect(destroy.exitCode).toBe(0);
        expect(destroy.stderr).toContain('Destroyed');
        expect(destroy.stderr).toContain('project world(s) destroyed');
        expect(destroy.container('neo').exists).toBe(false);

        // Then - list --json reports no running worlds
        await using list = await cli.fixture('$FIXTURES/docker-pilot/').exec('world list --json');

        expect(list.exitCode).toBe(0);
        const report = list.json.value as {
            mode: string;
            worlds: Array<{ name: string; status: string }>;
        };
        expect(report.mode).toBe('project');
        expect(report.worlds.every((w) => w.status !== 'running')).toBe(true);
    });

    test('world list is stable across repeated calls', async () => {
        // Given - a spawned world and its first list snapshot
        await using first = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'world list --json']);

        expect(first.exitCode).toBe(0);
        expect(first.container('neo').running).toBe(true);

        const firstList = first.json.value as {
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(firstList.worlds).toHaveLength(1);
        expect(firstList.worlds[0].status).toBe('running');

        // When - a second list runs while the first container is still live under the label
        await using second = await cli.fixture('$FIXTURES/docker-pilot/').exec('world list --json');

        // Then - the second snapshot matches the first
        expect(second.exitCode).toBe(0);
        const secondList = second.json.value as {
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(secondList.worlds).toHaveLength(1);
        expect(secondList.worlds[0]).toEqual(firstList.worlds[0]);
    });
});
