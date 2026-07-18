import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * World lifecycle under docker-aware mode. Exercises the full
 * ContainerAccessor surface: `.running`/`.status` from the post-run
 * inspect snapshot, `.file(path).exists` via docker exec, `.exec(cmd)`
 * returning a CliResult, plus the host-side CLI output asserts. Every
 * result binds with `await using` so containers are force-removed at scope
 * exit (rule B5).
 */
describe('world lifecycle', () => {
    test('up provisions a running world with agent files laid down', async () => {
        // Given - the docker-pilot world brought online
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

        // Then - progress banners land on stderr and the container is live
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Created container');
        expect(result.stderr).toContain('Agent is alive');

        const neo = result.container('neo');
        expect(neo.exists).toBe(true);
        expect(neo.running).toBe(true);
        expect(neo.status).toBe('running');
        expect(neo.file('/agents/neo/CLAUDE.md').exists).toBe(true);

        // And the agent runs as the unprivileged spwn user inside the box
        const whoami = await neo.exec('id -un');
        expect(whoami.exitCode).toBe(0);
        expect(whoami.stdout.text.trim()).toBe('spwn');
    });

    test('world list surfaces the running world in project mode', async () => {
        // Given - the world brought up then listed as JSON
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'world list --json']);

        // Then - one running project world with the neo agent, and it is live
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
        expect(result.container('neo').running).toBe(true);
    });

    test('world inspect renders the expected field headers', async () => {
        // Given - a running world whose id is resolved from .spwn/world-states
        await using inspect = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['up', 'world inspect $(ls .spwn/world-states 2>/dev/null | head -1)']);

        // Then - the stepper renders the core field headers (scalpel: third-party stepper layout, not a stable golden)
        expect(inspect.exitCode).toBe(0);
        expect(inspect.stderr).toContain('Status');
        expect(inspect.stderr).toContain('Agent home');
    });

    test('world enter validates the world before printing the banner', async () => {
        // Given - an initialised project with no running worlds, entered with a bogus id
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init', 'world enter nonexistent']);

        // Then - the success banner never prints and the error names the target (scalpel: absence + dynamic error surface)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).not.toContain('Entering');
        expect(result.stderr).toContain('nonexistent');
    });

    test('world enter --help renders the enter usage block', async () => {
        // Given - the enter help flag
        await using result = await cli.fixture('$FIXTURES/empty/').exec('world enter --help');

        // Then - cobra prints the full enter usage block (byte-for-byte golden)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('enter-help.txt');
    });

    test('down fully destroys the container (not just stopped)', async () => {
        // Given - up followed by down
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec(['up', 'down']);

        // Then - the teardown banners fire and the container is gone from docker
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Destroyed');
        expect(result.stderr).toContain('project world(s) destroyed');
        expect(result.container('neo').exists).toBe(false);
    });
});
