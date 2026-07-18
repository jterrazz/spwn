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
 * The happy-path tests pin `SPWN_BASE_IMAGE=spwn-test:latest` so the
 * container uses the mock runtime binary. Every result binds with
 * `await using` so the spawned container is force-removed at scope exit
 * (rule B5).
 */
describe('agent --ephemeral (NPC)', () => {
    test('ephemeral without --world fails with a helpful error', async () => {
        // Given - an ephemeral dispatched with no world specified
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('agent --ephemeral "do something"');

        // Then - exit 1 with a friendly error and no runtime crash noise (scalpel: message probe over dynamic wording)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).not.toContain('TypeError');
        expect(result.stderr).not.toContain('panic');
        expect(result.stderr).not.toContain('goroutine');
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/--world|world specified|active worlds/i);
    });

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
        expect(worldID).toBeTruthy();

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
        expect(worldID).toBeTruthy();

        // When - an ephemeral is dispatched and agents are listed in the same run
        await using after = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec([`agent --ephemeral "check health" --world ${worldID}`, 'agent ls --json']);

        // Then - only the declared neo agent exists; no ephemeral row was persisted
        expect(after.exitCode).toBe(0);
        const report = after.json.value as {
            agents: Array<{ name: string }>;
        };
        const names = report.agents.map((a) => a.name);
        expect(names).toContain('neo');
        expect(names).not.toContain('npc');
        expect(names).not.toContain('ephemeral');
    });
});
