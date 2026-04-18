import type { CliResult } from '@jterrazz/test';
import { afterAll, beforeAll, describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * World spawn under the docker() spec mode.
 *
 * Read-only assertions share one upped world via beforeAll/afterAll —
 * every file/label/inspect check against a post-`up` container is the
 * same shape and does not mutate state, so spawning once and running
 * seven assertions against it saves ~6× container create/start cost.
 * Tests that need their own flow (error paths, down, world list
 * chained with up) keep their own spec call.
 *
 * Dropped from legacy coverage (all intentional — CLI shapes changed):
 *   - "world ID format is w-{name}-{5digits}": current ids use
 *     `world-<planet>-<hex>`. Covered indirectly by the label
 *     assertion below.
 *   - "ID does not contain 'default'": there is no "default" config
 *     concept in project mode.
 *   - "workspace files visible inside container": covered by
 *     workspace.e2e.test.ts.
 *   - "default mode is detached": project-mode `spwn up` is always
 *     detached.
 *   - "bare `world up` (no --agent) spawns empty world": project mode
 *     always has an agent from the worlds: map.
 */
describe('world spawn', () => {
    describe('shared upped world', () => {
        let world: CliResult;

        beforeAll(async () => {
            world = await spec('world spawn shared').project('docker-pilot').exec('up').run();
            expect(world.exitCode).toBe(0);
        });

        afterAll(async () => {
            await world[Symbol.asyncDispose]();
        });

        test('up emits the expected stderr banners', () => {
            // Banner lines land on stderr per Unix convention.
            world.stderr.toContain('Created container');
            world.stderr.toContain('Agent is alive');
        });

        test('container is running with world/ files laid down', () => {
            const neo = world.container('neo');
            expect(neo.exists).toBe(true);
            expect(neo.running).toBe(true);
            expect(neo.status).toBe('running');

            expect(neo.file('/world/physics.md').exists).toBe(true);
            expect(neo.file('/world/faculties.md').exists).toBe(true);
        });

        test('container carries a valid world id label', () => {
            const inspectData = world.container('neo').inspect.value as {
                Config?: { Labels?: Record<string, string> };
            };
            const labels = inspectData.Config?.Labels ?? {};
            const worldId = labels['sh.spwn.world.id'];
            expect(worldId).toBeTruthy();
            expect(worldId).toMatch(/^world-[a-z0-9-]+-[0-9a-f]{5}$/);
            expect(labels['sh.spwn.world.config']).toBe('neo');
            expect(labels['sh.spwn.kind']).toBe('world');
        });

        test('physics.md carries meaningful content (laws, network, /workspaces)', () => {
            const physics = world.container('neo').file('/world/physics.md').content;
            expect(physics).toMatch(/Laws/);
            expect(physics).toMatch(/network/i);
            expect(physics).toMatch(/\/workspaces/);
        });

        test('faculties.md lists the expanded tool set', () => {
            const faculties = world.container('neo').file('/world/faculties.md').content;
            expect(faculties).toMatch(/spwn:unix/);
        });

        test('mind layers are visible at /agents/neo/ inside the container', async () => {
            const neo = world.container('neo');
            // identity/ was collapsed into SOUL.md at the agent root in
            // 2026-04. The Mind now has three directory layers plus the
            // soul file. Knowledge is world-scoped at /world/knowledge/.
            expect(neo.file('/agents/neo/SOUL.md').exists).toBe(true);
            for (const layer of ['skills', 'playbooks', 'journal']) {
                expect(neo.file(`/agents/neo/${layer}`).exists).toBe(true);
            }

            const ls = await neo.exec('ls /agents/neo');
            expect(ls.exitCode).toBe(0);
            ls.stdout.toContain('SOUL.md');
            for (const layer of ['skills', 'playbooks', 'journal']) {
                ls.stdout.toContain(layer);
            }

            // And the world-scoped knowledge base is where it should be.
            expect(neo.file('/world/knowledge').exists).toBe(true);
        });

        test('agent home has Claude trust pre-approved for its workspaces', () => {
            const neo = world.container('neo');

            // The runtime provider writes .claude.json into each agent's
            // Real HOME (/agents/<name>) at spawn time so the per-agent
            // Claude Code run drops straight into a ready state without
            // Onboarding prompts.
            const claudeJson = neo.file('/agents/neo/.claude.json').content;
            const config = JSON.parse(claudeJson) as {
                hasCompletedOnboarding?: boolean;
                projects?: Record<string, { hasTrustDialogAccepted?: boolean }>;
            };
            expect(config.hasCompletedOnboarding).toBe(true);
            expect(config.projects?.['/workspaces']?.hasTrustDialogAccepted).toBe(true);
            expect(config.projects?.['/agents/neo']?.hasTrustDialogAccepted).toBe(true);

            const settings = neo.file('/agents/neo/.claude/settings.json').content;
            const settingsConfig = JSON.parse(settings) as {
                skipDangerousModePermissionPrompt?: boolean;
            };
            expect(settingsConfig.skipDangerousModePermissionPrompt).toBe(true);
        });

        test('container does NOT bind-mount host .claude or /agents tree', () => {
            const neo = world.container('neo');

            // The agent's .claude/settings.json should be the minimal
            // Config spwn ships, not the host's (which has hooks,
            // Dependencies, etc.).
            const settings = neo.file('/agents/neo/.claude/settings.json').content;
            const settingsConfig = JSON.parse(settings) as {
                enabledPlugins?: unknown;
                hooks?: unknown;
            };
            expect(settingsConfig.hooks).toBeUndefined();
            expect(settingsConfig.enabledPlugins).toBeUndefined();

            // Inspect confirms nothing under /agents is a host bind
            // Mount — agent homes are docker-cp'd in at spawn time,
            // And memory layers are docker-cp'd back out on graceful
            // Down. No bind equals no host leak.
            const inspectData = neo.inspect.value as {
                Mounts?: Array<{ Destination: string }>;
            };
            const mounts = inspectData.Mounts ?? [];
            const agentMount = mounts.find(
                (m) =>
                    m.Destination === '/agents' ||
                    m.Destination.startsWith('/agents/') ||
                    m.Destination === '/home/spwn/.claude',
            );
            expect(agentMount).toBeUndefined();
        });
    });

    describe('per-test flows', () => {
        test('spawned world appears in `world list --json`', async () => {
            await using result = await spec('spawn in list')
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
            expect(list.worlds[0]).toEqual({
                agents: ['neo'],
                name: 'neo',
                status: 'running',
            });
        });

        test('spawn with an unknown world name fails cleanly with no container leaked', async () => {
            // In project mode, `spwn up <name>` resolves <name> against the
            // Worlds: map in spwn.yaml. Asking for a key that isn't there is
            // The closest analogue to the legacy "non-existent config" case.
            await using result = await spec('spawn unknown world')
                .project('docker-pilot')
                .exec('up nonexistent-world')
                .run();

            expect(result.exitCode).toBe(1);

            expect(result.stderr.text).not.toContain('panic');
            expect(result.stderr.text).not.toContain('goroutine');
            // Error message names the missing world and points users back
            // At spwn.yaml — this is the hint contract we lock down.
            await result.stderr.toMatch('spawn-unknown-world.txt');
            expect(result.container('neo').exists).toBe(false);
            expect(result.container('nonexistent-world').exists).toBe(false);
        });

        test('spawn with a non-existent agent fails cleanly', async () => {
            await using result = await spec('spawn missing agent')
                .project('docker-pilot')
                .exec('world up --agent ghost')
                .run();

            expect(result.exitCode).toBe(1);

            expect(result.stderr.text).not.toContain('panic');
            expect(result.stderr.text).not.toContain('goroutine');
            // Hint at `spwn agent new <name>` so users can fix it.
            // (Exact wording from apps/cli/world/world.go:456.)
            await result.stderr.toMatch('spawn-missing-agent.txt');
            expect(result.container('ghost').exists).toBe(false);
        });

        test('`spwn down` fully removes the container', async () => {
            await using result = await spec('spawn then down')
                .project('docker-pilot')
                .exec(['up', 'down'])
                .run();

            expect(result.exitCode).toBe(0);
            result.stderr.toContain('Destroyed');
            expect(result.container('neo').exists).toBe(false);
        });
    });
});
