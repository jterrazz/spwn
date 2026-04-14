import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * World security / physics enforcement under the docker() spec mode.
 *
 * Dropped (legacy assertions that no longer apply to current spwn):
 *   - "physics constants are documented in physics.md":
 *     The physics.md template no longer materialises CPU / Memory /
 *     Timeout values. Physics constants were removed from the manifest
 *     as part of a previous refactor; see resources.e2e.test.ts for
 *     the PidsLimit assertion that survived.
 *   - "element pack expansion with @spwn/node":
 *     The current seed handler surface can't override spwn/worlds/
 *     default.yaml. The docker-pilot fixture ships @spwn/unix and
 *     @spwn/git only. Testing @spwn/node would require a new fixture
 *     clone — out of scope for this migration. The unix + git
 *     expansion is exercised below.
 *
 * Preserved and augmented:
 *   - bash and git are reachable inside the container (element packs
 *     actually wire binaries in)
 *   - faculties.md lists the expanded tools
 *   - Default network mode is bridge
 *   - PidsLimit is bounded (see also resources.e2e.test.ts)
 */
describe('world security', () => {
    test('declared element packs expand into live binaries inside the container', async () => {
        await using result = await spec('element pack expansion')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');
        expect(neo.running).toBe(true);

        // The docker-pilot fixture's worlds/default.yaml declares
        // @spwn/unix + @spwn/git, so bash and git must resolve on $PATH.
        const bash = await neo.exec('which bash');
        expect(bash.exitCode).toBe(0);
        expect(bash.stdout.text).toContain('bash');

        const git = await neo.exec('which git');
        expect(git.exitCode).toBe(0);
        expect(git.stdout.text).toContain('git');

        // And faculties.md reflects the verified tool set.
        const faculties = neo.file('/world/faculties.md').content;
        expect(faculties).toMatch(/bash/);
        expect(faculties).toMatch(/git/);
    });

    test('default network mode is bridge', async () => {
        // Spwn currently runs world containers on the bridge network so
        // Agents can reach the host through host.docker.internal. This
        // Test pins the behaviour — flip it the day spwn reintroduces a
        // Network isolation flag.
        await using result = await spec('network bridge').project('docker-pilot').exec('up').run();

        expect(result.exitCode).toBe(0);

        const inspectData = result.container('neo').inspect.value as {
            HostConfig?: { NetworkMode?: string };
        };
        expect(inspectData.HostConfig?.NetworkMode).toBe('bridge');
    });

    test('pids limit is bounded (not unlimited)', async () => {
        await using result = await spec('pids limit bounded')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const inspectData = result.container('neo').inspect.value as {
            HostConfig?: { PidsLimit?: number };
        };
        const pidsLimit = inspectData.HostConfig?.PidsLimit ?? 0;
        // 0 and -1 both mean unlimited; a positive number means the
        // Backend wired the limit through from physics config.
        expect(pidsLimit).toBeGreaterThan(0);
    });
});
