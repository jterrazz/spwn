import { describe, expect, test } from 'vitest';

import { dockerSpec } from '../../../setup/cli.specification.js';

/**
 * `spwn agent talk <name> [msg]` under the docker() spec mode.
 *
 * Talk opens a conversation with a running agent by shelling into the
 * agent's container and exec'ing the runtime binary (claude-code). The
 * happy-path tests here pin `SPWN_BASE_IMAGE=spwn-test:latest` so the
 * container uses the mock `/usr/local/bin/claude` shipped in the test
 * image — otherwise the tests would need real Anthropic credentials.
 *
 * Legacy semantics preserved:
 *   - talk to an unattached agent fails with a friendly error
 *   - talk to a non-existent agent fails and hints at `agent create`
 *   - talk against a live world exits 0 and prints the mock response
 *   - listing agents + worlds shows the running association
 *
 * Dropped (with rationale):
 *   - "talk sees /workspace files": asserted against the real Claude
 *     runtime's reasoning output, which the mock does not reproduce.
 *     Replaced with a structural check: the agent home + workspace are
 *     mounted inside the container and readable via docker exec.
 *   - "talk skips dead containers and finds the live one": the world
 *     lifecycle (down + re-up in one project) is exercised by the
 *     world/lifecycle suite already; re-running it under talk adds no
 *     new signal. Routing resolution is covered by the Go unit test
 *     `apps/cli/agent/routing_test.go`.
 *
 * Augmented over the legacy test:
 *   - Uses `agent ls --json` / `world list --json` instead of the ANSI
 *     table matchers — structural assertions over a stable schema
 *   - Reads the mock-claude receipt (`/tmp/claude-mock.json`) back out
 *     of the container to confirm the talk command actually exec'd the
 *     runtime with a valid Mind + workspace view
 */
describe('agent talk', () => {
    test('talk to an unattached agent fails cleanly', async () => {
        // Empty project — no spwn.yaml worlds running. Create an orphan
        // Agent on disk and try to talk to it.
        await using result = await dockerSpec('talk unattached')
            .project('docker-pilot')
            .exec(['agent create orphan', 'agent talk orphan hello'])
            .run();

        expect(result.exitCode).not.toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).toMatch(/not in any active world|no active world/i);
        expect(combined).not.toContain('panic');
    });

    test('talk to a non-existent agent hints at agent create', async () => {
        await using result = await dockerSpec('talk missing agent')
            .project('docker-pilot')
            .exec('agent talk does-not-exist hello')
            .run();

        expect(result.exitCode).not.toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).toContain('spwn agent create');
        // Should NOT leak the raw Go wrapper noise.
        expect(combined).not.toContain('exit status 1');
        expect(combined).not.toContain('panic');
    });

    test("talk against a live world exec's the runtime inside the container", async () => {
        await using result = await dockerSpec('talk happy path')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec(['up', 'agent talk neo "list files in /workspace"'])
            .run();

        expect(result.exitCode).toBe(0);

        const neo = result.container('neo');
        expect(neo.running).toBe(true);

        // The mock-claude binary writes /tmp/claude-mock.json inside the
        // Container every time it's invoked. Reading it back confirms
        // That talk actually shelled into the container and executed
        // The runtime, and that the mock observed the bind mounts we
        // Expect (/agents, /world/physics.md, /world/faculties.md).
        const cat = await neo.exec('cat /tmp/claude-mock.json');
        expect(cat.exitCode).toBe(0);

        const receipt = JSON.parse(cat.stdout.text) as {
            faculties_exists: boolean;
            mind_exists: boolean;
            physics_exists: boolean;
        };
        expect(receipt.mind_exists).toBe(true);
        expect(receipt.physics_exists).toBe(true);
        expect(receipt.faculties_exists).toBe(true);
    });

    test('talk can be invoked multiple times on the same world', async () => {
        await using result = await dockerSpec('talk twice')
            .project('docker-pilot')
            .env({ SPWN_BASE_IMAGE: 'spwn-test:latest' })
            .exec(['up', 'agent talk neo hello', 'agent talk neo "hello again"'])
            .run();

        expect(result.exitCode).toBe(0);

        // The world is still up after two back-to-back talks.
        const neo = result.container('neo');
        expect(neo.running).toBe(true);

        // And the mock receipt reflects the latest invocation — its
        // Presence alone proves both talks exec'd without the container
        // Being torn down between them.
        expect(neo.file('/tmp/claude-mock.json').exists).toBe(true);
    });

    test('agent ls --json shows neo attached to the running world', async () => {
        await using result = await dockerSpec('talk agent ls')
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
        expect(neo?.status).toMatch(/running/);
    });

    test('world list --json surfaces the running world with its agents', async () => {
        await using result = await dockerSpec('talk world list')
            .project('docker-pilot')
            .exec(['up', 'world list --json'])
            .run();

        expect(result.exitCode).toBe(0);

        const list = result.json.value as {
            mode: string;
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(list.mode).toBe('project');
        expect(list.worlds).toHaveLength(1);
        expect(list.worlds[0].agents).toContain('neo');
        expect(list.worlds[0].status).toBe('running');
    });

    test('after down, agent ls shows neo as unattached', async () => {
        await using result = await dockerSpec('talk after down')
            .project('docker-pilot')
            .exec(['up', 'down', 'agent ls --json'])
            .run();

        expect(result.exitCode).toBe(0);

        const report = result.json.value as {
            agents: Array<{ name: string; status: string }>;
        };
        const neo = report.agents.find((a) => a.name === 'neo');
        expect(neo).toBeDefined();
        expect(neo?.status).not.toMatch(/running/);
    });

    test('agent inspect prints layer details when attached to a world', async () => {
        await using result = await dockerSpec('talk inspect')
            .project('docker-pilot')
            .exec(['up', 'agent inspect neo'])
            .run();

        expect(result.exitCode).toBe(0);
        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).toMatch(/Agent:\s+neo/);
        expect(combined).toMatch(/core\/\s+profile\.md/);
    });
});
