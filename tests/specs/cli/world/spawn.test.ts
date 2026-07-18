import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * World spawn under docker-aware mode. The v8 file shared one upped world
 * across eight read-only assertions via beforeAll/afterAll (a nude `let
 * world` — rule B5). Those assertions are consolidated into a single
 * cohesive test so the file still boots one container for the full
 * agent-home/label/inspect surface, now owned by `await using`. The
 * error/flow paths keep their own results.
 *
 * Dropped from v8 (CLI shapes changed): the `w-{name}-{5digits}` id format,
 * the "no default config" probe, workspace visibility (see workspace.test),
 * detached-by-default, and bare `world up` empty-world semantics.
 */
describe('world spawn', () => {
    test('up lays down the full agent home, world context, and container labels', async () => {
        // Given - a freshly-upped docker-pilot world
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

        // Then - the boot banners fire and the container is running
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Created container');
        expect(result.stderr).toContain('Agent is alive');
        const neo = result.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
        expect(neo.status).toBe('running');

        // World context is inlined into each agent's CLAUDE.md (scalpel: probing inlined markers)
        const claude = neo.file('/agents/neo/CLAUDE.md').content;
        expect(claude).toMatch(/Laws/);
        expect(claude).toMatch(/network/i);
        expect(claude).toMatch(/\/workspaces/);
        expect(claude).toMatch(/spwn:unix/);

        // The container carries a valid world id plus config/kind labels
        const inspectData = neo.inspect.value as {
            Config?: { Labels?: Record<string, string> };
            Mounts?: Array<{ Destination: string }>;
        };
        const labels = inspectData.Config?.Labels ?? {};
        expect(labels['sh.spwn.world.id']).toMatch(/^world-[a-z0-9-]+-[0-9a-f]{5}$/);
        expect(labels['sh.spwn.world.config']).toBe('neo');
        expect(labels['sh.spwn.kind']).toBe('world');

        // The mind layers are laid down at /agents/neo with world-scoped knowledge
        expect(neo.file('/agents/neo/SOUL.md').exists).toBe(true);
        for (const layer of ['playbooks', 'journal']) {
            expect(neo.file(`/agents/neo/${layer}`).exists).toBe(true);
        }
        const ls = await neo.exec('ls /agents/neo');
        expect(ls.exitCode).toBe(0);
        expect(ls.stdout).toContain('SOUL.md');
        for (const layer of ['playbooks', 'journal']) {
            expect(ls.stdout).toContain(layer);
        }
        expect(neo.file('/world/knowledge').exists).toBe(true);

        // Claude trust is pre-approved so the per-agent run drops straight into a ready state
        const claudeJson = neo.file('/agents/neo/.claude.json').content;
        const config = JSON.parse(claudeJson) as {
            hasCompletedOnboarding?: boolean;
            projects?: Record<string, { hasTrustDialogAccepted?: boolean }>;
        };
        expect(config.hasCompletedOnboarding).toBe(true);
        expect(config.projects?.['/workspaces']?.hasTrustDialogAccepted).toBe(true);
        expect(config.projects?.['/agents/neo']?.hasTrustDialogAccepted).toBe(true);

        // The shipped settings are minimal (no host hooks/plugins leaked in)
        const settings = neo.file('/agents/neo/.claude/settings.json').content;
        const settingsConfig = JSON.parse(settings) as {
            enabledPlugins?: unknown;
            hooks?: unknown;
            skipDangerousModePermissionPrompt?: boolean;
        };
        expect(settingsConfig.skipDangerousModePermissionPrompt).toBe(true);
        expect(settingsConfig.hooks).toBeUndefined();
        expect(settingsConfig.enabledPlugins).toBeUndefined();

        // And nothing under /agents is a host bind mount — homes are docker-cp'd in
        const mounts = inspectData.Mounts ?? [];
        const agentMount = mounts.find(
            (mount) =>
                mount.Destination === '/agents' ||
                mount.Destination.startsWith('/agents/') ||
                mount.Destination === '/home/spwn/.claude',
        );
        expect(agentMount).toBeUndefined();
    });

    test('spawned world appears in world list --json', async () => {
        // Given - the world upped then listed as JSON
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'world list --json']);

        // Then - one running project world with the neo agent
        expect(result.exitCode).toBe(0);
        const list = result.json.value as {
            mode: string;
            worlds: Array<{ agents: string[]; name: string; status: string }>;
        };
        expect(list.mode).toBe('project');
        expect(list.worlds).toHaveLength(1);
        expect(list.worlds[0]).toEqual({
            agents: ['neo'],
            name: 'neo',
            status: 'running',
        });
    });

    test('spawn with an unknown world name fails cleanly with no container leaked', async () => {
        // Given - a project-mode up requesting a world key that is not in spwn.yaml
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('up nonexistent-world');

        // Then - non-zero with the canonical hint (full stderr golden) and no container left behind
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('spawn-unknown-world.txt');
        expect(result.container('neo').exists).toBe(false);
        expect(result.container('nonexistent-world').exists).toBe(false);
    });

    test('spawn with a non-existent agent fails cleanly', async () => {
        // Given - a world up naming an agent that does not exist
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('world up --agent ghost');

        // Then - non-zero with the "agent new" hint (full stderr golden) and no container left behind
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('spawn-missing-agent.txt');
        expect(result.container('ghost').exists).toBe(false);
    });

    test('spwn down fully removes the container', async () => {
        // Given - up followed by down
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec(['up', 'down']);

        // Then - the teardown banner fires and the container is gone
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Destroyed');
        expect(result.container('neo').exists).toBe(false);
    });
});
