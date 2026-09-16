import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Ephemeral (NPC) agents under docker-aware mode.
 *
 * `spwn agent --ephemeral "<task>" --world <id>` dispatches a fire-and-
 * forget task into a running world. It runs `claude` (or the mocked test
 * binary) inside the container and then exits, leaving no Mind directory
 * behind and no new agent in `agent ls`.
 *
 * Both specs here spawn a container and read it back, which is why they
 * stay chains: they pin `SPWN_BASE_IMAGE=spwn-test:latest` so the world
 * uses the mock runtime, read the world id off the container labels, and
 * bind every result with `await using` so the container is force-removed
 * at scope exit (rule B5). The refusal path — an ephemeral with no world
 * — spawns nothing and is a document beside this file.
 */
describe('agent --ephemeral (NPC)', () => {
    test('ephemeral dispatches a task into a running world', async () => {
        // Given - a world brought up, whose runtime id is read off the container labels
        await using up = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('up');

        expect(up.exitCode).toBe(0);
        const neo = up.container('neo');
        expect(neo.running).toBe(true);
        const inspectData = neo.inspect.value as {
            Config?: { Labels?: Record<string, string> };
        };
        const worldID = inspectData.Config?.Labels?.['sh.spwn.world.id'];
        expect(worldID).toMatch(/^world-/u);

        // When - an ephemeral is dispatched against the live world
        await using dispatch = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec(`agent --ephemeral "lint the code" --world ${worldID}`);

        // Then - the dispatch succeeds and the host world survives it (scalpel: dispatch banner probe over dynamic output)
        expect(dispatch.exitCode).toBe(0);
        expect(dispatch.stdout.text + dispatch.stderr.text).toMatch(
            /Ephemeral dispatched|lint the code/,
        );
        expect(up.container('neo').running).toBe(true);
    });

    test('ephemeral does not register a persistent agent', async () => {
        // Given - a world brought up, whose runtime id is read off the container labels
        await using up = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('up');

        expect(up.exitCode).toBe(0);
        const labels =
            (up.container('neo').inspect.value as { Config?: { Labels?: Record<string, string> } })
                .Config?.Labels ?? {};
        const worldID = labels['sh.spwn.world.id'];
        expect(worldID).toMatch(/^world-/u);

        // When - an ephemeral is dispatched and agents are listed in the same run
        await using after = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec([`agent --ephemeral "check health" --world ${worldID}`, 'agent ls --json']);

        // Then - only the declared neo agent exists; no ephemeral row was persisted
        expect(after.exitCode).toBe(0);
        const report = after.json.value as {
            agents: { name: string }[];
        };
        const names = report.agents.map((a) => a.name);
        expect(names).toContain('neo');
        expect(names).not.toContain('npc');
        expect(names).not.toContain('ephemeral');
    });
});
