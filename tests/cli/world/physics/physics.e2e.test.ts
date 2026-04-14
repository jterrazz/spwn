import type { CliResult } from '@jterrazz/test';
import { afterAll, beforeAll, describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * World physics.
 *
 * Every test here is a read-only assertion on a freshly-upped world,
 * so we share ONE container across the file via beforeAll/afterAll.
 * Spawning once instead of per-test cuts ~4× container create/start
 * cost out of the run. The framework's test-run label isolates this
 * shared container from any other file running in parallel.
 *
 * Legacy coverage dropped:
 *   - "inspect shows physics constants" — `spwn world inspect` no
 *     longer surfaces a Constants block; the feature is gone.
 *   - "physics.md contains CPU/Memory/Timeout" — physics.md template
 *     does not materialise constants anymore.
 */
describe('world physics', () => {
    let world: CliResult;

    beforeAll(async () => {
        world = await spec('world physics shared').project('docker-pilot').exec('up').run();
        expect(world.exitCode).toBe(0);
    });

    afterAll(async () => {
        await world[Symbol.asyncDispose]();
    });

    test('physics.md and faculties.md exist inside the container', async () => {
        const neo = world.container('neo');
        expect(neo.running).toBe(true);

        expect(neo.file('/world/physics.md').exists).toBe(true);
        expect(neo.file('/world/faculties.md').exists).toBe(true);

        const ls = await neo.exec('ls /world');
        expect(ls.exitCode).toBe(0);
        ls.stdout.toContain('physics.md');
        ls.stdout.toContain('faculties.md');
    });

    test('physics.md documents the network law and topology', () => {
        const physics = world.container('neo').file('/world/physics.md').content;
        expect(physics).toMatch(/network/i);
        expect(physics).toMatch(/Laws/);
        expect(physics).toMatch(/\/workspace/);
    });

    test('faculties.md lists available tools', () => {
        const faculties = world.container('neo').file('/world/faculties.md').content;
        expect(faculties).toMatch(/bash/);
    });

    test('default network mode is bridge', () => {
        const inspectData = world.container('neo').inspect.value as {
            HostConfig?: { NetworkMode?: string };
        };
        expect(inspectData.HostConfig?.NetworkMode).toBe('bridge');
    });
});
