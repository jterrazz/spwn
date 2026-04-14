import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Agent "inside a running world" under the docker() spec mode.
 *
 * Legacy semantics preserved:
 *   - Agent Mind is mounted into the container at /agents/<name>/...
 *   - Spawn prints the "Agent is alive" banner and creates the container
 *   - Mind layers (core/skills/knowledge/journal) exist on disk
 *   - `spwn agent ls` / `spwn agent inspect neo` cross-reference the
 *     running world
 *
 * Augmented over the legacy test:
 *   - Reads /agents/neo and /agents/neo/core/profile.md directly from
 *     inside the container via the container accessor
 *   - Verifies the container is live post-spawn via the docker inspect
 *     snapshot (`container.running`)
 *
 * Dropped:
 *   - Regex on `a-neo-\d{5}` agent IDs in `spwn world inspect` — the
 *     world inspect command now takes a full container/world ID (not a
 *     world key) and the test framework owns world id generation through
 *     labels, so the legacy regex no longer describes anything stable.
 *     The structural check (agent visible inside the container, state
 *     tracks the agent) is preserved via `agent ls --json`.
 */
describe('agent inside world', () => {
    test('agent home is mounted at /agents/<name> inside the world container', async () => {
        await using result = await spec('agent mount').project('docker-pilot').exec('up').run();

        expect(result.exitCode).toBe(0);
        // Banners go to stderr.
        result.stderr.toContain('Created container');
        result.stderr.toContain('Agent is alive');

        const neo = result.container('neo');
        expect(neo.running).toBe(true);

        // The agent's home directory is bind-mounted read-write into
        // The container via the single /agents mount.
        expect(neo.file('/agents/neo').exists).toBe(true);
        expect(neo.file('/agents/neo/core').exists).toBe(true);
        expect(neo.file('/agents/neo/core/profile.md').exists).toBe(true);

        const ls = await neo.exec('ls /agents/neo');
        expect(ls.exitCode).toBe(0);
        ls.stdout.toContain('core');
    });

    test('spawn confirms the agent is alive and the container is running', async () => {
        await using result = await spec('agent alive banner')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Agent is alive');
        result.stderr.toContain('Created container');

        const neo = result.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
        expect(neo.status).toBe('running');
    });

    test('agent ls --json surfaces neo attached to the running world', async () => {
        await using result = await spec('agent ls running')
            .project('docker-pilot')
            .exec(['up', 'agent ls --json'])
            .run();

        expect(result.exitCode).toBe(0);

        const report = result.json.value as {
            agents: Array<{ name: string; status: string; world?: string }>;
            mode: string;
        };
        expect(report.mode).toBe('project');

        const neo = report.agents.find((a) => a.name === 'neo');
        expect(neo).toBeDefined();
        // In project mode, status is a decorated string like "● running (…)".
        expect(neo?.status).toMatch(/running/);
    });

    test('the on-disk Mind for neo has all expected layers', async () => {
        await using result = await spec('mind on disk').project('docker-pilot').exec('up').run();

        expect(result.exitCode).toBe(0);

        // Docker-pilot ships neo's Mind under spwn/agents/neo/ in the
        // Fixture, so the project-local on-disk layers must be present.
        expect(result.file('spwn/agents/neo/core/profile.md').exists).toBe(true);
        expect(result.file('spwn/agents/neo/skills').exists).toBe(true);
        expect(result.file('spwn/agents/neo/knowledge').exists).toBe(true);
        expect(result.file('spwn/agents/neo/journal').exists).toBe(true);

        // And the same profile.md is visible inside the container via
        // The /agents bind mount — round-trip check through docker exec.
        const neo = result.container('neo');
        const cat = await neo.exec('cat /agents/neo/core/profile.md');
        expect(cat.exitCode).toBe(0);
        expect(cat.stdout.text.length).toBeGreaterThan(0);
    });
});
