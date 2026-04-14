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
 *     `spwn-world-<planet>-<digits>`. Covered indirectly by the label
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
            expect(worldId).toMatch(/^(?:spwn-world|w)-[a-z0-9-]+-\d{5}$/);
            expect(labels['sh.spwn.world.config']).toBe('neo');
            expect(labels['sh.spwn.kind']).toBe('world');
        });

        test('physics.md carries meaningful content (laws, network, /workspace)', () => {
            const physics = world.container('neo').file('/world/physics.md').content;
            expect(physics).toMatch(/Laws/);
            expect(physics).toMatch(/network/i);
            expect(physics).toMatch(/\/workspace/);
        });

        test('faculties.md lists the expanded tool set', () => {
            const faculties = world.container('neo').file('/world/faculties.md').content;
            expect(faculties).toMatch(/bash/);
        });

        test('mind layers are visible at /agents/neo/ inside the container', async () => {
            const neo = world.container('neo');
            for (const layer of ['core', 'skills', 'knowledge', 'playbooks', 'journal']) {
                expect(neo.file(`/agents/neo/${layer}`).exists).toBe(true);
            }

            const ls = await neo.exec('ls /agents/neo');
            expect(ls.exitCode).toBe(0);
            for (const layer of ['core', 'skills', 'knowledge', 'playbooks', 'journal']) {
                ls.stdout.toContain(layer);
            }
        });

        test('container has Claude trust pre-approved for /workspace', () => {
            const neo = world.container('neo');

            const claudeJson = neo.file('/home/spwn/.claude.json').content;
            const config = JSON.parse(claudeJson) as {
                hasCompletedOnboarding?: boolean;
                projects?: Record<string, { hasTrustDialogAccepted?: boolean }>;
            };
            expect(config.hasCompletedOnboarding).toBe(true);
            expect(config.projects?.['/workspace']?.hasTrustDialogAccepted).toBe(true);

            const settings = neo.file('/home/spwn/.claude/settings.json').content;
            const settingsConfig = JSON.parse(settings) as {
                skipDangerousModePermissionPrompt?: boolean;
            };
            expect(settingsConfig.skipDangerousModePermissionPrompt).toBe(true);
        });

        test('container does NOT mount host ~/.claude directory', () => {
            const neo = world.container('neo');

            // The container settings.json should be the minimal config spwn
            // Ships, not the host's (which has hooks, plugins, etc.).
            const settings = neo.file('/home/spwn/.claude/settings.json').content;
            const settingsConfig = JSON.parse(settings) as {
                enabledPlugins?: unknown;
                hooks?: unknown;
            };
            expect(settingsConfig.hooks).toBeUndefined();
            expect(settingsConfig.enabledPlugins).toBeUndefined();

            // And inspect confirms /home/spwn/.claude is not a host bind mount.
            const inspectData = neo.inspect.value as {
                Mounts?: Array<{ Destination: string }>;
            };
            const mounts = inspectData.Mounts ?? [];
            const claudeMount = mounts.find((m) => m.Destination === '/home/spwn/.claude');
            expect(claudeMount).toBeUndefined();
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
