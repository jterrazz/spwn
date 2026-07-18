import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Full agent-lifecycle journey under docker-aware mode. Walks the happy path
 * from `up` to teardown, asserting the CLI output and the live docker view
 * agree at each step. `world inspect` / `world logs` take the spwn world id read
 * off the container's `sh.spwn.world.id` label. Each spawning result binds with
 * `await using` so its container is force-removed at scope exit (rule B5).
 */
describe('full agent lifecycle', () => {
    test('journey: up -> ls -> inspect -> logs -> down', async () => {
        // Given - a docker-pilot world brought up
        await using up = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

        // Then - the container is created and running, and carries a world id label
        expect(up.exitCode).toBe(0);
        expect(up.stderr).toContain('Created container');
        expect(up.stderr).toContain('Agent is alive');
        const neo = up.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
        const worldId = (neo.inspect.value as { Config?: { Labels?: Record<string, string> } })
            .Config?.Labels?.['sh.spwn.world.id'];
        expect(worldId).toBeTruthy();

        // When - agent ls reports neo as running
        await using agentLs = await cli.fixture('$FIXTURES/docker-pilot/').exec('agent ls --json');

        // Then - the project-mode report lists neo running
        expect(agentLs.exitCode).toBe(0);
        const agentReport = agentLs.json.value as {
            agents: Array<{ name: string; status: string }>;
            mode: string;
        };
        expect(agentReport.mode).toBe('project');
        const neoAgent = agentReport.agents.find((a) => a.name === 'neo');
        expect(neoAgent).toBeDefined();
        expect(neoAgent?.status).toMatch(/running/);

        // When - world list --json reports one running world
        await using worldLs = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('world list --json');

        // Then - exactly one running project world attaches neo
        expect(worldLs.exitCode).toBe(0);
        const worldReport = worldLs.json.value as {
            mode: string;
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(worldReport.mode).toBe('project');
        expect(worldReport.worlds).toHaveLength(1);
        expect(worldReport.worlds[0]).toEqual({ agents: ['neo'], name: 'neo', status: 'running' });

        // When - world inspect <id> renders stable field headers
        await using inspect = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(`world inspect ${worldId}`);

        // Then - the stepper echoes the world id and a Status field (scalpel: stepper output on stderr)
        expect(inspect.exitCode).toBe(0);
        expect(inspect.stderr).toContain(worldId!);
        expect(inspect.stderr).toContain('Status');

        // When - world logs <id> is read
        await using logs = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(`world logs ${worldId}`);

        // Then - it exits cleanly
        expect(logs.exitCode).toBe(0);

        // When - the world is torn down
        await using down = await cli.fixture('$FIXTURES/docker-pilot/').exec(['up', 'down']);

        // Then - the destroy banners fire and the container is gone
        expect(down.exitCode).toBe(0);
        expect(down.stderr).toContain('Destroyed');
        expect(down.stderr).toContain('project world(s) destroyed');
        expect(down.container('neo').exists).toBe(false);
    });

    test('evolution: dream, sleep, fork, export', async () => {
        // Given - a chain of dream/sleep/fork/export against neo
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec([
                'agent dream neo',
                'agent sleep neo',
                'agent fork neo neo-v2',
                'agent export neo',
            ]);

        // Then - no crash, fork wrote a second on-disk mind, and export dropped a tarball at the project root
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('FATAL');
        expect(result.file('spwn/agents/neo-v2').exists).toBe(true);
        expect(result.file('spwn/agents/neo-v2/SOUL.md').exists).toBe(true);
        expect(result.file('neo.tar.gz').exists).toBe(true);
    });

    test('error recovery: operations on a deleted agent fail without crashing', async () => {
        // Given - neo removed then inspected
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['agent rm neo', 'agent inspect neo']);

        // Then - a clean non-zero exit, no panic or stack trace (scalpel: crash-signal absence)
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
        expect(result.stderr.text).not.toContain('FATAL');
        expect(result.stderr).toContain('neo');
    });

    test('error recovery: down on an invalid world id fails gracefully', async () => {
        // Given - a destroy against a fabricated world id
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('down world-fake-99999');

        // Then - a clean non-zero exit, no crash (scalpel: crash-signal absence)
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
        expect(result.stderr.text).not.toContain('FATAL');
    });

    test('error recovery: double destroy is idempotent', async () => {
        // Given - up then two downs in one chain
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'down', 'down']);

        // Then - no crash and the container is gone (scalpel: crash-signal absence)
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
        expect(result.stderr.text).not.toContain('FATAL');
        expect(result.container('neo').exists).toBe(false);
    });
});
