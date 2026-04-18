import type { CliResult } from '@jterrazz/test';
import { afterAll, beforeAll, describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Agent "inside a running world".
 *
 * Every test here inspects the same post-`up` state from a different
 * Angle — mount visibility, banner output, state reporting, on-disk
 * Mind layers. Share one container across the file.
 *
 * Dropped legacy assertion: regex on `a-neo-\d{5}` agent IDs in
 * `spwn world inspect`. That command now takes a full container id
 * And agent IDs aren't part of its output anymore; structural checks
 * Via `agent ls --json` preserve the intent.
 */
describe('agent inside world', () => {
    let world: CliResult;

    beforeAll(async () => {
        world = await spec('agent inside-world shared').project('docker-pilot').exec('up').run();
        expect(world.exitCode).toBe(0);
    });

    afterAll(async () => {
        await world[Symbol.asyncDispose]();
    });

    test('up emits Created container / Agent is alive banners on stderr', () => {
        world.stderr.toContain('Created container');
        world.stderr.toContain('Agent is alive');
    });

    test('agent home is mounted at /agents/<name> inside the world container', async () => {
        const neo = world.container('neo');
        expect(neo.running).toBe(true);

        // Identity/ was collapsed into SOUL.md; the agent's soul now
        // Lives at /agents/<name>/SOUL.md and the Mind layer dirs sit
        // Alongside it (skills/, playbooks/, journal/).
        expect(neo.file('/agents/neo').exists).toBe(true);
        expect(neo.file('/agents/neo/SOUL.md').exists).toBe(true);
        expect(neo.file('/agents/neo/skills').exists).toBe(true);

        const ls = await neo.exec('ls /agents/neo');
        expect(ls.exitCode).toBe(0);
        ls.stdout.toContain('SOUL.md');
    });

    test('spawn confirms the agent is alive and the container is running', () => {
        const neo = world.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
        expect(neo.status).toBe('running');
    });

    test('agent ls --json surfaces neo attached to the running world', async () => {
        // Spec calls from the same file share the runner's test-run id,
        // So this follow-up sees the shared container spawned in beforeAll.
        // NOTE: intentionally NOT `await using` — `spwn agent ls` doesn't
        // Spawn containers, and the dispose hook would force-remove every
        // Container tagged with this file's shared test-run id, including
        // The one beforeAll created.
        const lsResult = await spec('agent ls call')
            .project('docker-pilot')
            .exec('agent ls --json')
            .run();

        expect(lsResult.exitCode).toBe(0);

        const report = lsResult.json.value as {
            agents: Array<{ name: string; status: string; world?: string }>;
            mode: string;
        };
        expect(report.mode).toBe('project');

        const neo = report.agents.find((a) => a.name === 'neo');
        expect(neo).toBeDefined();
        expect(neo?.status).toMatch(/running/);
    });

    test('the on-disk Mind for neo has all expected layers', async () => {
        // Docker-pilot ships neo's Mind under spwn/agents/neo/ in the
        // Fixture; the on-disk layers must be present in the shared workdir.
        // Knowledge is world-scoped now — no longer a Mind layer.
        expect(world.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        expect(world.file('spwn/agents/neo/skills').exists).toBe(true);
        expect(world.file('spwn/agents/neo/journal').exists).toBe(true);

        // And the same profile.md is visible inside the container via
        // The /agents bind mount — round-trip through docker exec.
        const neo = world.container('neo');
        const cat = await neo.exec('cat /agents/neo/SOUL.md');
        expect(cat.exitCode).toBe(0);
        expect(cat.stdout.text.length).toBeGreaterThan(0);
    });
});
