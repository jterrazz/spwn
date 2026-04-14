import { describe, expect, test } from 'vitest';

import { dockerSpec } from '../../../setup/cli.specification.js';

/**
 * Ephemeral (NPC) agents under the docker() spec mode.
 *
 * `spwn agent --ephemeral "<task>" --world <id>` dispatches a fire-and-
 * forget task into a running world. It runs `claude` (or the mocked
 * test binary) inside the container and then exits, leaving no Mind
 * directory behind and no new agent in `agent ls`.
 *
 * Because the ephemeral flow actually shells out to the runtime binary
 * inside the container, these tests pin SPWN_BASE_IMAGE to the prebuilt
 * `spwn-test:latest` image where `/usr/local/bin/claude` is a mock that
 * accepts the same flags and exits 0 — otherwise we'd need real Claude
 * credentials just to exercise the spawn path.
 *
 * Legacy semantics preserved:
 *   - ephemeral without --world fails cleanly (no stack traces)
 *   - ephemeral dispatched against a running world succeeds
 *   - ephemeral does not register a new agent in `agent ls`
 *
 * Augmented over the legacy test:
 *   - Reaches into the container and confirms /world/AGENT.md was
 *     materialised (the NPC context file architect writes before exec)
 *   - Asserts the target container is still running after the ephemeral
 *     command exits
 */
describe('agent --ephemeral (NPC)', () => {
    test('ephemeral without --world fails with a helpful error', async () => {
        await using result = await dockerSpec('ephemeral no world')
            .project('docker-pilot')
            .exec('agent --ephemeral "do something"')
            .run();

        expect(result.exitCode).not.toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');
        expect(combined).toMatch(/--world|world specified|active worlds/i);
    });

    test('ephemeral dispatches a task into a running world', async () => {
        // First run: bring up the world. Keep it alive for a follow-up
        // Second dockerSpec call so we can read the runtime world id
        // Off the container label map.
        await using up = await dockerSpec('ephemeral up')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('up')
            .run();

        expect(up.exitCode).toBe(0);

        const neo = up.container('neo');
        expect(neo.running).toBe(true);

        const inspectData = neo.inspect.value as {
            Config?: { Labels?: Record<string, string> };
        };
        const worldID = inspectData.Config?.Labels?.['sh.spwn.world.id'];
        expect(worldID).toBeTruthy();

        // Second run: dispatch the ephemeral against the live world.
        await using dispatch = await dockerSpec('ephemeral dispatch')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec(`agent --ephemeral "lint the code" --world ${worldID}`)
            .run();

        expect(dispatch.exitCode).toBe(0);
        const combined = `${dispatch.stdout.text}\n${dispatch.stderr.text}`;
        expect(combined).toMatch(/Ephemeral dispatched|lint the code/);

        // And the container still exists and is running after dispatch —
        // Ephemerals must not kill their host world.
        expect(up.container('neo').running).toBe(true);

        // And architect wrote the NPC AGENT.md context file into the
        // Container before exec'ing the runtime command.
        expect(up.container('neo').file('/world/AGENT.md').exists).toBe(true);
    });

    test('ephemeral does not register a persistent agent', async () => {
        // Bring up the world and learn its id.
        await using up = await dockerSpec('ephemeral no mind up')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec('up')
            .run();

        expect(up.exitCode).toBe(0);

        const labels =
            (up.container('neo').inspect.value as { Config?: { Labels?: Record<string, string> } })
                .Config?.Labels ?? {};
        const worldID = labels['sh.spwn.world.id'];
        expect(worldID).toBeTruthy();

        // Dispatch an ephemeral and then list agents in the same run, so
        // `agent ls --json` sees whatever the dispatch left behind.
        await using after = await dockerSpec('ephemeral then ls')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec([`agent --ephemeral "check health" --world ${worldID}`, 'agent ls --json'])
            .run();

        expect(after.exitCode).toBe(0);

        const report = after.json.value as {
            agents: Array<{ name: string }>;
        };
        // Only the declared neo agent exists; no "npc"/"ephemeral" row.
        const names = report.agents.map((a) => a.name);
        expect(names).toContain('neo');
        expect(names).not.toContain('npc');
        expect(names).not.toContain('ephemeral');
    });
});
