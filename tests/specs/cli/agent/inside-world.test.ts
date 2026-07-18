import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Agent "inside a running world".
 *
 * The legacy suite shared one `up` container across five tests via a
 * nude-assigned `beforeAll` (rule B5 violation). The read-only mount /
 * banner / on-disk-layer probes are consolidated here into one cohesive
 * boot; the roster view keeps its own chained boot. Every result binds
 * with `await using` so the container is force-removed at scope exit.
 *
 * Dropped legacy assertion: regex on `a-neo-\d{5}` agent IDs in `spwn
 * world inspect` — that command now takes a full container id and agent
 * IDs aren't part of its output anymore; the JSON roster preserves intent.
 */
describe('agent inside world', () => {
    test('up brings neo online with its mounts and on-disk mind', async () => {
        // Given - a world brought up from docker-pilot
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

        // Then - the up banners fire and the container is running (scalpel: banner presence probes)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Created container');
        expect(result.stderr).toContain('Agent is alive');
        const neo = result.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
        expect(neo.status).toBe('running');

        // And - the agent home is mounted at /agents/neo with its Soul and layer dirs
        expect(neo.file('/agents/neo').exists).toBe(true);
        expect(neo.file('/agents/neo/SOUL.md').exists).toBe(true);
        expect(neo.file('/agents/neo/playbooks').exists).toBe(true);
        const ls = await neo.exec('ls /agents/neo');
        expect(ls.exitCode).toBe(0);
        expect(ls.stdout).toContain('SOUL.md');

        // And - the on-disk mind layers exist in the workdir and round-trip through the bind mount
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.file('spwn/agents/neo/playbooks').exists).toBe(true);
        expect(result.file('spwn/agents/neo/journal').exists).toBe(true);
        const cat = await neo.exec('cat /agents/neo/SOUL.md');
        expect(cat.exitCode).toBe(0);
        expect(cat.stdout.text.length).toBeGreaterThan(0);
    });

    test('agent ls --json surfaces neo attached to the running world', async () => {
        // Given - a world brought up, then listed as JSON in one chain
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'agent ls --json']);

        // Then - neo is reported running under a project-mode roster (scalpel: structural probe over dynamic status)
        expect(result.exitCode).toBe(0);
        const report = result.json.value as {
            agents: Array<{ name: string; status: string; world?: string }>;
            mode: string;
        };
        expect(report.mode).toBe('project');
        const neo = report.agents.find((a) => a.name === 'neo');
        expect(neo).toBeDefined();
        expect(neo?.status).toMatch(/running/);
    });
});
