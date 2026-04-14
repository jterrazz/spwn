import type { CliResult } from '@jterrazz/test';
import { afterAll, beforeAll, describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * World security / physics enforcement.
 *
 * Every test is a read-only inspection of a fresh-up world, so we
 * Share a single container across the file.
 *
 * Dropped legacy coverage:
 *   - "physics constants documented in physics.md": the physics.md
 *     template no longer materialises CPU / Memory / Timeout values.
 *   - "element pack expansion with @spwn/node": the fixture ships
 *     @spwn/unix + @spwn/git only. Testing @spwn/node would require
 *     a new fixture clone — out of scope.
 */
describe('world security', () => {
    let world: CliResult;

    beforeAll(async () => {
        world = await spec('world security shared').project('docker-pilot').exec('up').run();
        expect(world.exitCode).toBe(0);
    });

    afterAll(async () => {
        await world[Symbol.asyncDispose]();
    });

    test('declared element packs expand into live binaries inside the container', async () => {
        const neo = world.container('neo');
        expect(neo.running).toBe(true);

        // Docker-pilot's worlds/default.yaml declares @spwn/unix + @spwn/git.
        const bash = await neo.exec('which bash');
        expect(bash.exitCode).toBe(0);
        expect(bash.stdout.text).toContain('bash');

        const git = await neo.exec('which git');
        expect(git.exitCode).toBe(0);
        expect(git.stdout.text).toContain('git');

        const faculties = neo.file('/world/faculties.md').content;
        expect(faculties).toMatch(/bash/);
        expect(faculties).toMatch(/git/);
    });

    test('default network mode is bridge', () => {
        // Spwn currently runs world containers on the bridge network so
        // Agents can reach the host via host.docker.internal. Flip this
        // The day spwn reintroduces a network isolation flag.
        const inspectData = world.container('neo').inspect.value as {
            HostConfig?: { NetworkMode?: string };
        };
        expect(inspectData.HostConfig?.NetworkMode).toBe('bridge');
    });

    test('pids limit is bounded (not unlimited)', () => {
        const inspectData = world.container('neo').inspect.value as {
            HostConfig?: { PidsLimit?: number };
        };
        const pidsLimit = inspectData.HostConfig?.PidsLimit ?? 0;
        // 0 / -1 mean unlimited; a positive number means physics config
        // Wired the limit through.
        expect(pidsLimit).toBeGreaterThan(0);
    });
});
