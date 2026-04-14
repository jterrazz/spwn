import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * State tracking under the docker() spec mode.
 *
 * The legacy state tests asserted on a project-local `.spwn/state.json`
 * file that no longer exists — in project mode the source of truth is
 * docker label queries surfaced via `world list --json`, and the legacy
 * `StatePath()` is a user-level fallback for non-project mode. Rather
 * than fabricating a file path, these tests exercise the observable
 * contract the legacy suite meant to cover:
 *
 *   - after `up`, the world appears in `world list --json` and in docker
 *   - `down` removes the world from both views
 *   - `world list` is stable across repeated calls inside one run
 *
 * Augmented over the legacy test:
 *   - Cross-checks each assertion with the live docker container status
 *     via `result.container('neo').running` / `.exists`
 *   - Reads the world status structurally from JSON rather than parsing
 *     ANSI-decorated ls output
 *
 * Dropped:
 *   - "multiple worlds tracked in state": in project mode every declared
 *     world in spwn.yaml is spawned together by `up`. The legacy test
 *     spawned two independent worlds sharing an agent through the old
 *     ad-hoc CLI; that path no longer exists. Multi-world tracking is
 *     exercised by the config / multi-world tests under `cli/config/`.
 *
 * Note on multi-exec chains: only the *last* command's stdout/stderr is
 * captured. To assert on banners from an earlier step (e.g. "Created
 * container"), split the chain into separate spec calls and keep
 * the earlier result in scope with `await using` so its container is
 * still live under the test-run label for follow-up queries.
 */
describe('state tracking', () => {
    test('after up, banners fire and world list --json reports the world', async () => {
        await using up = await spec('state up banner').project('docker-pilot').exec('up').run();

        expect(up.exitCode).toBe(0);
        up.stderr.toContain('Created container');
        up.stderr.toContain('Agent is alive');

        const neo = up.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);

        // Now query list from a second spec — our container is still
        // In scope under the test-run label so it's visible there too.
        await using list = await spec('state up list')
            .project('docker-pilot')
            .exec('world list --json')
            .run();

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
        // Chain up → down so the last command's streams carry the
        // Destroy banners; world list --json is queried separately
        // Below, but the destroy banner assertions need to be on the
        // Run whose last step is `down`.
        await using destroy = await spec('state down banner')
            .project('docker-pilot')
            .exec(['up', 'down'])
            .run();

        expect(destroy.exitCode).toBe(0);
        destroy.stderr.toContain('Destroyed');
        destroy.stderr.toContain('project world(s) destroyed');

        // Container gone from docker entirely
        expect(destroy.container('neo').exists).toBe(false);

        // And list --json reports no running worlds
        await using list = await spec('state down list')
            .project('docker-pilot')
            .exec('world list --json')
            .run();

        expect(list.exitCode).toBe(0);
        const report = list.json.value as {
            mode: string;
            worlds: Array<{ name: string; status: string }>;
        };
        expect(report.mode).toBe('project');
        expect(report.worlds.every((w) => w.status !== 'running')).toBe(true);
    });

    test('world list is stable across repeated calls', async () => {
        // First run: spawn the world and grab list #1.
        await using first = await spec('state list stable first')
            .project('docker-pilot')
            .exec(['up', 'world list --json'])
            .run();

        expect(first.exitCode).toBe(0);
        expect(first.container('neo').running).toBe(true);

        const firstList = first.json.value as {
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(firstList.worlds).toHaveLength(1);
        expect(firstList.worlds[0].status).toBe('running');

        // Second run inside the same `await using` scope: the first
        // Container is still live under the test-run label, so a new
        // `world list --json` in a fresh spec reads the same snapshot.
        await using second = await spec('state list stable second')
            .project('docker-pilot')
            .exec('world list --json')
            .run();

        expect(second.exitCode).toBe(0);
        const secondList = second.json.value as {
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(secondList.worlds).toHaveLength(1);
        expect(secondList.worlds[0]).toEqual(firstList.worlds[0]);
    });
});
