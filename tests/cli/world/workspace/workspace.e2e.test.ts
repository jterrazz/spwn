import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * World workspace persistence under the docker() spec mode.
 *
 * Legacy semantics preserved:
 *   - Files on the host are visible inside the container
 *   - Files created in the container persist back to the host
 *   - The workspace mount is read-write
 *   - Spawning with a non-existent workspace path fails without a stack trace
 *
 * Augmented over the legacy test:
 *   - Asserts the bind mount shape via `container('neo').inspect` (source +
 *     target + rw mode) so a regression in the mount wiring is caught
 *     directly rather than via an indirect file write
 */
describe('world workspace', () => {
    test('workspace is bind-mounted read-write into the container', async () => {
        await using result = await spec('workspace bind mount')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const neo = result.container('neo');
        expect(neo.running).toBe(true);

        // Inspect the container's binds: docker-pilot declares workspaces: [.]
        // So the project root should be mounted at /work/default read-write.
        const inspectData = neo.inspect.value as {
            HostConfig?: { Binds?: string[] };
            Mounts?: Array<{ Destination: string; Mode?: string; RW?: boolean; Source: string }>;
        };

        const mounts = inspectData.Mounts ?? [];
        const workspaceMount = mounts.find((m) => m.Destination === '/work/default');
        expect(workspaceMount).toBeDefined();
        expect(workspaceMount?.RW).toBe(true);
        // The mount source should be on the host filesystem (the fresh
        // Project directory spwn is running in).
        expect(workspaceMount?.Source).toBeTruthy();
    });

    test('host project files are visible inside the container', async () => {
        await using result = await spec('host to container visibility')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const neo = result.container('neo');
        // The spwn.yaml file lives at the root of the docker-pilot fixture
        // That was copied into the fresh project dir, so it must be visible
        // In the mounted workspace inside the container.
        expect(neo.file('/work/default/spwn.yaml').exists).toBe(true);
        const content = neo.file('/work/default/spwn.yaml').content;
        expect(content).toContain('docker-pilot');
    });

    test('files written in the container persist to the host workspace', async () => {
        await using result = await spec('container to host persistence')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const neo = result.container('neo');

        // Given - a file created inside the container
        const write = await neo.exec(
            'sh -c "echo \'created in container\' > /work/default/container-file.txt"',
        );
        expect(write.exitCode).toBe(0);

        // Then - the file is present on the host workspace (workDir), which
        // The result.file() accessor reads directly from disk.
        const hostFile = result.file('container-file.txt');
        expect(hostFile.exists).toBe(true);
        expect(hostFile.content.trim()).toBe('created in container');
    });

    test('workspace mount is read-write (write then read back inside container)', async () => {
        await using result = await spec('workspace rw roundtrip')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const neo = result.container('neo');

        const write = await neo.exec(
            'sh -c "echo \'rw-test-content\' > /work/default/rw-test.txt"',
        );
        expect(write.exitCode).toBe(0);

        const read = await neo.exec('cat /work/default/rw-test.txt');
        expect(read.exitCode).toBe(0);
        expect(read.stdout.text.trim()).toBe('rw-test-content');
    });

    test('up with a non-existent workspace path fails gracefully', async () => {
        await using result = await spec('bad workspace path')
            .project('docker-pilot')
            .seed('spwn.yaml/bad-workspace.yaml')
            .exec('up')
            .run();

        expect(result.exitCode).not.toBe(0);

        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        // No stack trace or panic — spwn should surface a clean error
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');

        // And no container was left behind for the neo world
        expect(result.container('neo').exists).toBe(false);
    });
});
