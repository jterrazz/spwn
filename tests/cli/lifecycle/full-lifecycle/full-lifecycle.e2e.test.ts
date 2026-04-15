import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Full agent-lifecycle end-to-end journey under the docker() spec mode.
 *
 * Walks through the happy-path a user takes from `up` to teardown,
 * asserting at each step that both the CLI output and the live docker
 * view agree on what happened.
 *
 * Legacy semantics preserved:
 *   - up → ls → inspect → logs → down → dream → sleep → fork → export
 *   - fork creates a second on-disk mind
 *   - export writes a tar.gz next to the project
 *   - error recovery: operations on deleted agents are clean
 *
 * Adapted for project mode:
 *   - `spwn up` replaces `spwn up --agent neo -w <workspace>`; docker-pilot
 *     declares neo in spwn.yaml so bare up is the right call
 *   - `spwn world inspect` / `world logs` take a full container id; we
 *     read it off `container('neo').id` (captured via test-run label)
 *   - `spwn ls` is the agent-centric shortcut; world list --json is used
 *     for structural assertions
 *
 * Dropped:
 *   - `spwn agent rm` at the very end of the happy path: the docker-pilot
 *     fixture is discarded when the spec's temp dir is cleaned up, so
 *     removing the agents adds no new signal. The error-recovery tests
 *     below already exercise `agent rm` + operations on deleted agents.
 *
 * Multi-exec chains only expose the last command's streams, so the
 * journey is split into several spec calls that keep the world
 * alive under the test-run label across calls.
 */
describe('full agent lifecycle', () => {
    test('journey: up → ls → inspect → logs → down', async () => {
        // Step 1: up (also checks banners on stderr)
        await using up = await spec('journey up').project('docker-pilot').exec('up').run();

        expect(up.exitCode).toBe(0);
        up.stderr.toContain('Created container');
        up.stderr.toContain('Agent is alive');

        const neo = up.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);

        // `spwn world inspect` / `world logs` expect the *spwn world id*
        // (The label we stamp at create time), not the docker container
        // Id. Read it off the label map.
        const inspectData = neo.inspect.value as {
            Config?: { Labels?: Record<string, string> };
        };
        const worldId = inspectData.Config?.Labels?.['sh.spwn.world.id'];
        expect(worldId).toBeTruthy();

        // Step 2: agent ls — neo shows up as running
        await using agentLs = await spec('journey agent ls')
            .project('docker-pilot')
            .exec('agent ls --json')
            .run();

        expect(agentLs.exitCode).toBe(0);
        const agentReport = agentLs.json.value as {
            agents: Array<{ name: string; status: string }>;
            mode: string;
        };
        expect(agentReport.mode).toBe('project');
        const neoAgent = agentReport.agents.find((a) => a.name === 'neo');
        expect(neoAgent).toBeDefined();
        expect(neoAgent?.status).toMatch(/running/);

        // Step 3: world list --json — one running world
        await using worldLs = await spec('journey world list')
            .project('docker-pilot')
            .exec('world list --json')
            .run();

        expect(worldLs.exitCode).toBe(0);
        const worldReport = worldLs.json.value as {
            mode: string;
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(worldReport.mode).toBe('project');
        expect(worldReport.worlds).toHaveLength(1);
        expect(worldReport.worlds[0]).toEqual({
            agents: ['neo'],
            name: 'neo',
            status: 'running',
        });

        // Step 4: world inspect <id> — stable field headers
        await using inspect = await spec('journey inspect')
            .project('docker-pilot')
            .exec(`world inspect ${worldId}`)
            .run();

        expect(inspect.exitCode).toBe(0);
        // `world inspect` renders via stepper on stderr.
        inspect.stderr.toContain(worldId!);
        expect(inspect.stderr.text).toMatch(/Status/);

        // Step 5: world logs <id> — must not crash
        await using logs = await spec('journey logs')
            .project('docker-pilot')
            .exec(`world logs ${worldId}`)
            .run();

        expect(logs.exitCode).toBe(0);

        // Step 6: down — destroy banners + container removed
        await using down = await spec('journey down')
            .project('docker-pilot')
            .exec(['up', 'down'])
            .run();

        expect(down.exitCode).toBe(0);
        down.stderr.toContain('Destroyed');
        down.stderr.toContain('project world(s) destroyed');
        expect(down.container('neo').exists).toBe(false);
    });

    test('evolution: dream, sleep, fork, export', async () => {
        await using result = await spec('journey evolve')
            .project('docker-pilot')
            .exec([
                'agent dream neo',
                'agent sleep neo',
                'agent fork neo neo-v2',
                'agent export neo',
            ])
            .run();

        // Each of these may exit 0 on success; the combined exit code
        // Propagates the last failing step if any. Assert the journey
        // Produced the expected on-disk side-effects regardless of the
        // Exact last-step streams captured.
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('FATAL');

        // Fork wrote a second on-disk mind.
        expect(result.file('spwn/agents/neo-v2').exists).toBe(true);
        expect(result.file('spwn/agents/neo-v2/identity/profile.md').exists).toBe(true);

        // Export dropped a tar.gz somewhere in the project — the legacy
        // Test checked for neo.tar.gz at the project root.
        expect(result.file('neo.tar.gz').exists).toBe(true);
    });

    test('error recovery: operations on a deleted agent fail without crashing', async () => {
        await using result = await spec('journey deleted agent')
            .project('docker-pilot')
            .exec(['agent rm neo', 'agent inspect neo'])
            .run();

        // Inspect on a missing agent should exit 1 with a clean
        // Error — no panics, no Go stack traces.
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
        expect(result.stderr.text).not.toContain('FATAL');
        result.stderr.toContain('neo');
    });

    test('error recovery: down on an invalid world id fails gracefully', async () => {
        await using result = await spec('journey bad id')
            .project('docker-pilot')
            .exec('down w-fake-99999')
            .run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
        expect(result.stderr.text).not.toContain('FATAL');
    });

    test('error recovery: double destroy is idempotent', async () => {
        await using result = await spec('journey double destroy')
            .project('docker-pilot')
            .exec(['up', 'down', 'down'])
            .run();

        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');
        expect(result.stderr.text).not.toContain('FATAL');
        expect(result.container('neo').exists).toBe(false);
    });
});
