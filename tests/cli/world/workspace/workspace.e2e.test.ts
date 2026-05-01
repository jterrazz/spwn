import type { CliResult } from '@jterrazz/test';
import { afterAll, beforeAll, describe, expect, test } from 'vitest';

import { spec } from '../../../_setup/cli.specification.js';

/**
 * World workspace persistence under the docker() spec mode.
 *
 * Read-only / independent-write tests share one container. Each
 * Writing test targets a distinct filename in the resolved
 * /workspaces/<name> mount so they don't conflict. The
 * bad-workspace-path error case spins up its own spec since it tests
 * a failing `up`.
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

        const workspaceRoot = () => {
            const neo = world.container('neo');
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
            const workspaceMount = mounts.find((m) => m.Destination.startsWith('/workspaces/'));
            expect(workspaceMount, `mounts: ${JSON.stringify(mounts)}`).toBeDefined();
            if (!workspaceMount) {
                throw new Error('workspace mount not found');
            }
            return workspaceMount;
        };

        test('workspace is bind-mounted read-write into the container', () => {
            const neo = world.container('neo');
            expect(neo.running).toBe(true);

            // Docker-pilot declares workspaces: [.] so the project root is
            // Mounted under /workspaces/<resolved-name> read-write. The
            // Name depends on the temp project basename and can be
            // Workspace0 or a slugified fixture/test label.
            const workspaceMount = workspaceRoot();
            expect(workspaceMount).toBeDefined();
            expect(workspaceMount?.RW).toBe(true);
            expect(workspaceMount?.Source).toBeTruthy();
        });

        test('host project files are visible inside the container', () => {
            const neo = world.container('neo');
            const mount = workspaceRoot();
            // The spwn.yaml file lives at the root of the docker-pilot fixture.
            expect(neo.file(`${mount.Destination}/spwn.yaml`).exists).toBe(true);
            const content = neo.file(`${mount.Destination}/spwn.yaml`).content;
            expect(content).toContain('docker-pilot');
        });

        test('files written in the container persist to the host workspace', async () => {
            const neo = world.container('neo');
            const mount = workspaceRoot();

            // Unique filename so other shared tests never race this one.
            const write = await neo.exec(
                `sh -c ${JSON.stringify(`echo 'created in container' > ${mount.Destination}/persist-test.txt`)}`,
            );
            expect(write.exitCode).toBe(0);

            const hostFile = world.file('persist-test.txt');
            expect(hostFile.exists).toBe(true);
            expect(hostFile.content.trim()).toBe('created in container');
        });

        test('workspace mount is read-write (write then read back inside container)', async () => {
            const neo = world.container('neo');
            const mount = workspaceRoot();

            // Distinct filename from the persist test above.
            const write = await neo.exec(
                `sh -c ${JSON.stringify(`echo 'rw-test-content' > ${mount.Destination}/rw-test.txt`)}`,
            );
            expect(write.exitCode).toBe(0);

            const read = await neo.exec(`cat ${mount.Destination}/rw-test.txt`);
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

        expect(result.exitCode).toBe(1);

        expect(result.stderr.text).not.toContain('TypeError');
        expect(result.stderr.text).not.toContain('panic');
        expect(result.stderr.text).not.toContain('goroutine');

        expect(result.container('neo').exists).toBe(false);
    });
});
