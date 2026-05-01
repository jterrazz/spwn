import { describe, expect, test } from 'vitest';

import { spec } from '../../../_setup/cli.specification.js';

/**
 * World lifecycle under the docker() spec mode.
 *
 * Exercises the full assertion surface of ContainerAccessor:
 *   - `.running` / `.status` from the post-run docker inspect snapshot
 *   - `.file(path).exists` via `docker exec test -e` inside the container
 *   - `.exec(cmd)` returning a CliResult over `docker exec sh -c`
 *   - top-level CLI output asserts for the host-side command (exit code,
 *     stdout substrings) — same `toContain` API as CLI-only tests
 *
 * Cleanup is automatic: `await using` force-removes every container
 * tagged with this test's run id when the scope exits.
 */
describe('world lifecycle', () => {
    test('up provisions a running world with world/ files laid down', async () => {
        await using result = await spec('up lifecycle').project('docker-pilot').exec('up').run();

        expect(result.exitCode).toBe(0);
        // Spwn writes progress banners to stderr (Unix convention) and
        // The World/Agent summary to stdout at exit time.
        result.stderr.toContain('Created container');
        result.stderr.toContain('Agent is alive');

        const neo = result.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
        expect(neo.status).toBe('running');

        // World context is inlined into CLAUDE.md now; no separate
        // /world/physics.md or /world/faculties.md files.
        expect(neo.file('/agents/neo/CLAUDE.md').exists).toBe(true);

        const whoami = await neo.exec('id -un');
        expect(whoami.exitCode).toBe(0);
        expect(whoami.stdout.text.trim()).toBe('spwn');
    });

    test('world list surfaces the running world in project mode', async () => {
        await using result = await spec('list after up')
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

        // And the container is live
        expect(result.container('neo').running).toBe(true);
    });

    test('world inspect renders the expected field headers', async () => {
        // Given - a running project world
        // When - `spwn world inspect <id>` runs against it
        // Then - the rendered output includes the core field headers
        // (Status, Agent home). This locks down apps/cli/world/inspect.go's
        // Top-level field surface, which could silently regress now that
        // The Constants/Laws block is gone along with physics config.
        //
        // World id is resolved inside the spec (single shared workdir)
        // Via shell substitution against .spwn/world-states/ — each
        // Subdir there is named after the world id.
        await using inspect = await spec('inspect fields')
            .project('docker-pilot')
            .exec(['up', 'world inspect $(ls .spwn/world-states 2>/dev/null | head -1)'])
            .run();
        expect(inspect.exitCode).toBe(0);
        // `world inspect` uses a Stepper that renders to stderr.
        expect(inspect.stderr.text).toMatch(/Status/);
        expect(inspect.stderr.text).toMatch(/Agent home/);
    });

    test('world enter validates the world before printing the banner', async () => {
        // Given - an initialised project with no running worlds
        // When - world enter is called with a bogus id
        // Then - the ✓ Entering success banner must never be printed
        const result = await spec('enter missing')
            .project('empty')
            .exec(['init', 'world enter nonexistent'])
            .run();

        expect(result.exitCode).toBe(1);
        // The success banner must not have been printed before the error.
        expect(result.stderr.text).not.toContain('Entering');
        // The error surface should say what we were trying to enter.
        expect(result.stderr.text).toMatch(/nonexistent/);
    });

    test('down fully destroys the container (not just stopped)', async () => {
        await using result = await spec('up then down')
            .project('docker-pilot')
            .exec(['up', 'down'])
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('Destroyed');
        result.stderr.toContain('project world(s) destroyed');

        // After destroy the container is gone from docker entirely, so the
        // Post-run label lookup finds nothing — `exists` is the right check.
        expect(result.container('neo').exists).toBe(false);
    });
});
