import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Colony (one world, two agents) — messaging + roster semantics under
 * the docker() spec mode.
 *
 * The sibling `colony/multi-agent` file already covers the basic "up
 * spawns two agents + container mounts" path. This file focuses on the
 * colony-specific behaviours the legacy test exercised on top of that:
 *
 *   - `/world/roster.md` exists inside the container and names both agents
 *   - `spwn agent send` delivers a message from one colony member to
 *     another; the JSON file lands in the target's in-container inbox
 *   - `spwn agent inbox <name>` surfaces the delivered messages
 *   - multiple messages from the same sender all persist and show up
 *   - `spwn down` tears the colony down cleanly; both agent minds
 *     survive on disk so a follow-up `up` can resurrect them
 *
 * Seeds:
 *   - agent/morpheus: second agent mind alongside docker-pilot's neo
 *   - spwn.yaml/colony.yaml: re-declares worlds.neo with [morpheus, neo]
 *
 * Dropped from the legacy file:
 *   - Assertions on role-tagged roster lines ("chief"/"worker"):
 *     `GenerateRoster` (packages/world/internal/physics/agent_context.go)
 *     defaults both agents to "worker" here, so the chief/worker
 *     distinction the legacy test asserted on is not meaningful in this
 *     fixture shape — agents get their roles from spwn.yaml role fields
 *     which colony.yaml doesn't set.
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

        // Roster.md is regenerated on every spawn from the live agent
        // Set. Assert both colony members are listed.
        const roster = neo.file('/world/roster.md').content;
        expect(roster).toContain('neo');
        expect(roster).toContain('morpheus');
        expect(roster).toContain('Roster');
    });

    test('agent send writes a message file into the target agent inbox', async () => {
        await using result = await spec('colony send')
            .project('docker-pilot')
            .seed('agent/morpheus')
            .seed('spwn.yaml/colony.yaml')
            .exec(['up', 'agent send neo --from morpheus "implement auth module"'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.stderr.text).toMatch(/Sent message\s+morpheus → neo/);

        // The message file lands in /world/inbox/neo/ inside the container.
        const neo = result.container('neo');
        expect(neo.file('/world/inbox/neo').exists).toBe(true);

        const ls = await neo.exec('ls /world/inbox/neo');
        expect(ls.exitCode).toBe(0);
        ls.stdout.toContain('.json');

        // And the JSON body carries the sender + content.
        const cat = await neo.exec('sh -c "cat /world/inbox/neo/*.json"');
        expect(cat.exitCode).toBe(0);
        const body = cat.stdout.text;
        expect(body).toContain('"from": "morpheus"');
        expect(body).toContain('"to": "neo"');
        expect(body).toContain('"content": "implement auth module"');
    });

    test('agent inbox surfaces a message delivered inside the colony', async () => {
        await using result = await spec('colony inbox')
            .project('docker-pilot')
            .seed('agent/morpheus')
            .seed('spwn.yaml/colony.yaml')
            .exec([
                'up',
                'agent send neo --from morpheus "implement auth module"',
                'agent inbox neo',
            ])
            .run();

        expect(result.exitCode).toBe(0);
        // Inbox renders a table on stderr (ui.Table default writer).
        result.stderr.toContain('morpheus');
        result.stderr.toContain('implement auth module');
        result.stderr.toContain('FROM');
    });

    test('multiple messages from chief to worker all persist', async () => {
        await using result = await spec('colony inbox multi')
            .project('docker-pilot')
            .seed('agent/morpheus')
            .seed('spwn.yaml/colony.yaml')
            .exec([
                'up',
                'agent send neo --from morpheus "task 1: setup database"',
                'agent send neo --from morpheus "task 2: implement API"',
                'agent send neo --from morpheus "task 3: write tests"',
                'agent inbox neo',
            ])
            .run();

        expect(result.exitCode).toBe(0);
        // Inbox table rows land on stderr.
        result.stderr.toContain('task 1: setup database');
        result.stderr.toContain('task 2: implement API');
        result.stderr.toContain('task 3: write tests');

        // And three JSON files live in the container inbox.
        const wc = await result.container('neo').exec('sh -c "ls /world/inbox/neo/*.json | wc -l"');
        expect(wc.exitCode).toBe(0);
        expect(wc.stdout.text.trim()).toBe('3');
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
        expect(result.file('spwn/agents/neo/identity/profile.md').exists).toBe(true);
        expect(result.file('spwn/agents/morpheus/identity/profile.md').exists).toBe(true);
    });
});
