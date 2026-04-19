import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Colony (one world, two agents) — roster + teardown semantics under
 * the docker() spec mode.
 *
 * The sibling `colony/multi-agent` file already covers the basic "up
 * spawns two agents + container mounts" path. This file focuses on the
 * colony-specific behaviours the legacy test exercised on top of that:
 *
 *   - `each agent's CLAUDE.md inlines a roster that names both agents
 *   - `spwn down` tears the colony down cleanly; both agent minds
 *     survive on disk so a follow-up `up` can resurrect them
 *
 * Seeds:
 *   - agent/morpheus: second agent mind alongside docker-pilot's neo
 *   - spwn.yaml/colony.yaml: re-declares worlds.neo with [morpheus, neo]
 *
 * Dropped from the legacy file:
 *   - `agent send`/`agent inbox` assertions: the mailbox-backed
 *     messaging commands were retired in the pre-1.0 cleanup. When a
 *     new messaging primitive lands it will get its own suite.
 *   - Assertions on role-tagged roster lines ("chief"/"worker"):
 *     `GenerateRoster` defaults both agents to "worker" here, so the
 *     chief/worker distinction the legacy test asserted on is not
 *     meaningful in this fixture shape — agents get their roles from
 *     spwn.yaml role fields which colony.yaml doesn't set.
 *   - `spwn ls` output-format assertions: covered in cli/execution
 *     and cli/colony/multi-agent `world list --json`.
 */
describe('colony', () => {
    test('roster.md inside the container names both colony members', async () => {
        await using result = await spec('colony roster')
            .project('docker-pilot')
            .seed('agent/morpheus')
            .seed('spwn.yaml/colony.yaml')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Created container');

        const neo = result.container('neo');
        expect(neo.running).toBe(true);

        // Roster used to live at /world/roster.md; it's now inlined
        // Into each agent's CLAUDE.md under "## Roster". Assert both
        // Colony members are listed in neo's prompt.
        const claude = neo.file('/agents/neo/CLAUDE.md').content;
        expect(claude).toMatch(/## Roster/);
        expect(claude).toContain('neo');
        expect(claude).toContain('morpheus');
    });

    test('destroying the colony removes the container but both agent minds survive', async () => {
        await using result = await spec('colony down survive')
            .project('docker-pilot')
            .seed('agent/morpheus')
            .seed('spwn.yaml/colony.yaml')
            .exec(['up', 'down'])
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Destroyed');
        result.stderr.toContain('project world(s) destroyed');

        // Container gone.
        expect(result.container('neo').exists).toBe(false);

        // Both on-disk minds survive teardown so a follow-up up can
        // Resurrect them. This matches the legacy "agent survives"
        // Contract the colony destroy test was asserting on.
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.file('spwn/agents/morpheus/SOUL.md').exists).toBe(true);
    });
});
