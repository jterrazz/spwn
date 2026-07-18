import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * World workspace persistence under docker-aware mode. The v8 file shared
 * one container across four read/write assertions via beforeAll/afterAll
 * (a nude `let world` — rule B5). Those are consolidated into a single
 * cohesive test so the file boots one container for the whole host↔
 * container sync surface, owned by `await using`. The bad-workspace-path
 * error case spins its own result from a fixture overlay.
 */

test('workspace is bind-mounted read-write and syncs between host and container', async () => {
    // Given - a freshly-upped docker-pilot world (declares workspaces: [.])
    await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

    // Then - the container is running with the project root bind-mounted under /workspaces
    expect(result.exitCode).toBe(0);
    const neo = result.container('neo');
    expect(neo.running).toBe(true);

    const inspectData = neo.inspect.value as {
        Mounts?: Array<{ Destination: string; RW?: boolean; Source: string }>;
    };
    const mounts = inspectData.Mounts ?? [];
    const mount = mounts.find((entry) => entry.Destination.startsWith('/workspaces/'));
    expect(mount, `mounts: ${JSON.stringify(mounts)}`).toBeDefined();
    if (!mount) {
        throw new Error('workspace mount not found');
    }
    expect(mount.RW).toBe(true);
    expect(mount.Source).toBeTruthy();

    // Host project files are visible inside the container
    expect(neo.file(`${mount.Destination}/spwn.yaml`).exists).toBe(true);
    expect(neo.file(`${mount.Destination}/spwn.yaml`).content).toContain('docker-pilot');

    // Files written in the container persist back to the host workspace
    const persist = await neo.exec(
        `sh -c ${JSON.stringify(`echo 'created in container' > ${mount.Destination}/persist-test.txt`)}`,
    );
    expect(persist.exitCode).toBe(0);
    const hostFile = result.file('persist-test.txt');
    expect(hostFile.exists).toBe(true);
    expect(hostFile.content.trim()).toBe('created in container');

    // And the mount is read-write when written and read back inside the container
    const write = await neo.exec(
        `sh -c ${JSON.stringify(`echo 'rw-test-content' > ${mount.Destination}/rw-test.txt`)}`,
    );
    expect(write.exitCode).toBe(0);
    const read = await neo.exec(`cat ${mount.Destination}/rw-test.txt`);
    expect(read.exitCode).toBe(0);
    expect(read.stdout.text.trim()).toBe('rw-test-content');
});

test('up with a non-existent workspace path fails gracefully', async () => {
    // Given - docker-pilot with an overlay pointing worlds.neo at a missing host path
    await using result = await cli
        .fixture('$FIXTURES/docker-pilot/')
        .fixture('bad-workspace/')
        .exec('up');

    // Then - non-zero with no crash and no container leaked (scalpel: absence probes on an error path)
    expect(result.exitCode).toBe(1);
    expect(result.stderr).not.toContain('TypeError');
    expect(result.stderr).not.toContain('panic');
    expect(result.stderr).not.toContain('goroutine');
    expect(result.container('neo').exists).toBe(false);
});
