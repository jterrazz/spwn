import { describe, expect, test } from 'vitest';

import { dockerSpec } from '../../../setup/cli.specification.js';

/**
 * World physics under the docker() spec mode.
 *
 * Legacy semantics preserved (where still applicable):
 *   - physics.md mentions network law
 *   - faculties.md is generated inside the container (lists bash)
 *   - /world directory structure is present (physics.md + faculties.md)
 *   - Default network mode is bridge
 *
 * Dropped:
 *   - "inspect shows physics constants": `spwn world inspect` now takes
 *     a full container ID (not a world key) and the output no longer
 *     carries a "Constants: CPU/Memory/Timeout" section at all — the
 *     assertion would match nothing on current spwn.
 *   - "physics.md contains CPU/Memory/Timeout": the current physics.md
 *     template documents Laws, Tools, Communication and Topology; there
 *     is no CPU/Memory/Timeout section anywhere. Constants are not
 *     materialised into the in-container file.
 *
 * Augmented over the legacy test:
 *   - Reads /world/physics.md and /world/faculties.md directly through
 *     the container file accessor rather than via a helper
 *   - Asserts the /world directory listing via an in-container exec
 */
describe('world physics', () => {
    test('physics.md and faculties.md exist inside the container', async () => {
        await using result = await dockerSpec('world files in container')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const neo = result.container('neo');
        expect(neo.running).toBe(true);

        // Both world files are materialised
        expect(neo.file('/world/physics.md').exists).toBe(true);
        expect(neo.file('/world/faculties.md').exists).toBe(true);

        // Directory listing matches expectations
        const ls = await neo.exec('ls /world');
        expect(ls.exitCode).toBe(0);
        ls.stdout.toContain('physics.md');
        ls.stdout.toContain('faculties.md');
    });

    test('physics.md documents the network law and topology', async () => {
        await using result = await dockerSpec('physics.md content')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const physics = result.container('neo').file('/world/physics.md').content;
        // Current spwn physics template: Laws + Topology sections
        expect(physics).toMatch(/network/i);
        expect(physics).toMatch(/Laws/);
        expect(physics).toMatch(/\/workspace/);
    });

    test('faculties.md lists available tools', async () => {
        await using result = await dockerSpec('faculties.md content')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const faculties = result.container('neo').file('/world/faculties.md').content;
        expect(faculties).toMatch(/bash/);
    });

    test('default network mode is bridge', async () => {
        await using result = await dockerSpec('network mode bridge')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const inspectData = result.container('neo').inspect.value as {
            HostConfig?: { NetworkMode?: string };
        };
        expect(inspectData.HostConfig?.NetworkMode).toBe('bridge');
    });
});
