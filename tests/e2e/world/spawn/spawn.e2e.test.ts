import { describe, expect, test } from 'vitest';

import { dockerSpec } from '../../../setup/cli.specification.js';

/**
 * World spawn under the docker() spec mode.
 *
 * Project-mode `spwn up` replaces the legacy global `spwn world --agent neo
 * -w <home>` invocation. Most legacy assertions survive once rewired
 * to the project flow; a few had to be dropped or re-scoped:
 *
 * Dropped:
 *   - "world ID format is w-{name}-{5digits}": current ids use the
 *     `spwn-world-<planet>-<digits>` format. Legacy pattern no longer
 *     matches. Covered indirectly by "id is present on labels" below.
 *   - "ID does not contain 'default'": there is no concept of a default
 *     config in project mode — worlds are keyed by the `worlds:` map
 *     entry name. No longer meaningful.
 *   - "workspace files visible inside container": fully covered by
 *     workspace.e2e.test.ts.
 *   - "default mode is detached": project-mode `spwn up` is always
 *     detached; the --interactive flag it checked against has no
 *     project-mode equivalent and the "Talk: spwn agent talk neo"
 *     hint wording is from the legacy single-world path.
 *   - "bare `world up` (no --agent) spawns an empty world": project
 *     mode always has an agent (from the worlds: map). The legacy
 *     behaviour only existed for global-mode ephemeral worlds.
 *
 * Preserved + augmented:
 *   - Created container exists, runs, carries /world/{physics,faculties}.md
 *   - physics.md contains meaningful content (Laws, network, /workspace)
 *   - faculties.md lists expanded tools (bash)
 *   - Spawned world surfaces in `world list --json` under project mode
 *   - World id label is present on the container
 *   - Non-existent config fails cleanly, no panic, no container leak
 *   - Non-existent agent fails cleanly
 *   - `spwn down` fully removes the container
 *   - Mind layers are visible inside the container at /agents/<name>/
 *   - Claude trust is pre-approved inside the container
 *   - Container does NOT mount the host ~/.claude directory
 */
describe('world spawn', () => {
    test('brings up a running container with world/ files laid down', async () => {
        await using result = await dockerSpec('spawn up basic')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);
        // Banner lines land on stderr per Unix convention.
        result.stderr.toContain('Created container');
        result.stderr.toContain('Agent is alive');

        const neo = result.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
        expect(neo.status).toBe('running');

        expect(neo.file('/world/physics.md').exists).toBe(true);
        expect(neo.file('/world/faculties.md').exists).toBe(true);
    });

    test('container carries a valid world id label', async () => {
        await using result = await dockerSpec('spawn id label')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const inspectData = result.container('neo').inspect.value as {
            Config?: { Labels?: Record<string, string> };
        };
        const labels = inspectData.Config?.Labels ?? {};
        const worldId = labels['sh.spwn.world.id'];
        expect(worldId).toBeTruthy();
        expect(worldId).toMatch(/^(?:spwn-world|w)-[a-z0-9-]+-\d{5}$/);
        // And the config label records the worlds: map entry.
        expect(labels['sh.spwn.world.config']).toBe('neo');
        // Kind label is world (not architect).
        expect(labels['sh.spwn.kind']).toBe('world');
    });

    test('spawned world appears in `world list --json`', async () => {
        await using result = await dockerSpec('spawn in list')
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

    test('physics.md carries meaningful content (laws, network, /workspace)', async () => {
        await using result = await dockerSpec('spawn physics content')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const physics = result.container('neo').file('/world/physics.md').content;
        expect(physics).toMatch(/Laws/);
        expect(physics).toMatch(/network/i);
        expect(physics).toMatch(/\/workspace/);
    });

    test('faculties.md lists the expanded tool set', async () => {
        await using result = await dockerSpec('spawn faculties content')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const faculties = result.container('neo').file('/world/faculties.md').content;
        expect(faculties).toMatch(/bash/);
    });

    test('spawn with an unknown world name fails cleanly with no container leaked', async () => {
        // In project mode, `spwn up <name>` resolves <name> against the
        // Worlds: map in spwn.yaml. Asking for a key that isn't there is
        // The closest analogue to the legacy "non-existent config" case.
        await using result = await dockerSpec('spawn unknown world')
            .project('docker-pilot')
            .exec('up nonexistent-world')
            .run();

        expect(result.exitCode).not.toBe(0);

        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        // No stack trace / panic / goroutine dump.
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');
        // No container with our label stuck around.
        expect(result.container('neo').exists).toBe(false);
        expect(result.container('nonexistent-world').exists).toBe(false);
    });

    test('spawn with a non-existent agent fails cleanly', async () => {
        await using result = await dockerSpec('spawn missing agent')
            .project('docker-pilot')
            .exec('world up --agent ghost')
            .run();

        expect(result.exitCode).not.toBe(0);

        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');
        // Error mentions the missing agent by name.
        expect(combined).toContain('ghost');
        // No container leaked.
        expect(result.container('ghost').exists).toBe(false);
    });

    test('`spwn down` fully removes the container', async () => {
        await using result = await dockerSpec('spawn then down')
            .project('docker-pilot')
            .exec(['up', 'down'])
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Destroyed');
        expect(result.container('neo').exists).toBe(false);
    });

    test('mind layers are visible at /agents/neo/ inside the container', async () => {
        await using result = await dockerSpec('spawn mind layers')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const neo = result.container('neo');
        for (const layer of ['core', 'skills', 'knowledge', 'playbooks', 'journal']) {
            expect(neo.file(`/agents/neo/${layer}`).exists).toBe(true);
        }

        // And a listing confirms all five show up in one call.
        const ls = await neo.exec('ls /agents/neo');
        expect(ls.exitCode).toBe(0);
        for (const layer of ['core', 'skills', 'knowledge', 'playbooks', 'journal']) {
            ls.stdout.toContain(layer);
        }
    });

    test('container has Claude trust pre-approved for /workspace', async () => {
        await using result = await dockerSpec('spawn claude trust')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');

        const claudeJson = neo.file('/home/spwn/.claude.json').content;
        const config = JSON.parse(claudeJson) as {
            hasCompletedOnboarding?: boolean;
            projects?: Record<string, { hasTrustDialogAccepted?: boolean }>;
        };
        expect(config.hasCompletedOnboarding).toBe(true);
        expect(config.projects?.['/workspace']?.hasTrustDialogAccepted).toBe(true);

        // And settings.json skips the dangerous mode permission prompt.
        const settings = neo.file('/home/spwn/.claude/settings.json').content;
        const settingsConfig = JSON.parse(settings) as {
            skipDangerousModePermissionPrompt?: boolean;
        };
        expect(settingsConfig.skipDangerousModePermissionPrompt).toBe(true);
    });

    test('container does NOT mount host ~/.claude directory', async () => {
        await using result = await dockerSpec('spawn no host claude mount')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');

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
