import type { CliResult } from '@jterrazz/test';
import { afterAll, beforeAll, describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * World workspace persistence under the docker() spec mode.
 *
 * Read-only / independent-write tests share one container. Each
 * Writing test targets a distinct filename in /work/default so they
 * don't conflict. The bad-workspace-path error case spins up its own
 * spec since it tests a failing `up`.
 *
 * Legacy semantics preserved:
 *   - Files on the host are visible inside the container
 *   - Files created in the container persist back to the host
 *   - The workspace mount is read-write
 *   - Spawning with a non-existent workspace path fails cleanly
 */
describe('world workspace', () => {
    describe('shared upped world', () => {
        let world: CliResult;

        beforeAll(async () => {
            world = await spec('world workspace shared').project('docker-pilot').exec('up').run();
            expect(world.exitCode).toBe(0);
        });

        afterAll(async () => {
            await world[Symbol.asyncDispose]();
        });

        test('workspace is bind-mounted read-write into the container', () => {
            const neo = world.container('neo');
            expect(neo.running).toBe(true);

            // Docker-pilot declares workspaces: [.] so the project root is
            // Mounted at /work/default read-write.
            const inspectData = neo.inspect.value as {
                HostConfig?: { Binds?: string[] };
                Mounts?: Array<{
                    Destination: string;
                    Mode?: string;
                    RW?: boolean;
                    Source: string;
                }>;
            };

            const mounts = inspectData.Mounts ?? [];
            const workspaceMount = mounts.find((m) => m.Destination === '/work/default');
            expect(workspaceMount).toBeDefined();
            expect(workspaceMount?.RW).toBe(true);
            expect(workspaceMount?.Source).toBeTruthy();
        });

        test('host project files are visible inside the container', () => {
            const neo = world.container('neo');
            // The spwn.yaml file lives at the root of the docker-pilot fixture.
            expect(neo.file('/work/default/spwn.yaml').exists).toBe(true);
            const content = neo.file('/work/default/spwn.yaml').content;
            expect(content).toContain('docker-pilot');
        });

        test('files written in the container persist to the host workspace', async () => {
            const neo = world.container('neo');

            // Unique filename so other shared tests never race this one.
            const write = await neo.exec(
                'sh -c "echo \'created in container\' > /work/default/persist-test.txt"',
            );
            expect(write.exitCode).toBe(0);

            const hostFile = world.file('persist-test.txt');
            expect(hostFile.exists).toBe(true);
            expect(hostFile.content.trim()).toBe('created in container');
        });

        test('workspace mount is read-write (write then read back inside container)', async () => {
            const neo = world.container('neo');

            // Distinct filename from the persist test above.
            const write = await neo.exec(
                'sh -c "echo \'rw-test-content\' > /work/default/rw-test.txt"',
            );
            expect(write.exitCode).toBe(0);

            const read = await neo.exec('cat /work/default/rw-test.txt');
            expect(read.exitCode).toBe(0);
            expect(read.stdout.text.trim()).toBe('rw-test-content');
        });
    });

    test('up with a non-existent workspace path fails gracefully', async () => {
        await using result = await spec('bad workspace path')
            .project('docker-pilot')
            .seed('spwn.yaml/bad-workspace.yaml')
            .exec('up')
            .run();

        expect(result.exitCode).not.toBe(0);

        const combined = `${result.stdout.text}\n${result.stderr.text}`;
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('panic');
        expect(combined).not.toContain('goroutine');

        expect(result.container('neo').exists).toBe(false);
    });
});
