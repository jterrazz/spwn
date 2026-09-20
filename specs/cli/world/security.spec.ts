import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * World security / physics enforcement. The v8 file shared one container
 * across three read-only assertions via beforeAll/afterAll (a nude `let
 * world` — rule B5). Those are consolidated into a single cohesive test so
 * the file boots exactly one container, owned by `await using`.
 *
 * Legacy coverage dropped: physics.md no longer materialises
 * CPU/Memory/Timeout values, and the fixture ships spwn:unix + spwn:git
 * only (a spwn:node expansion case would need a new fixture — out of scope).
 */

test('declared deps expand into live binaries and the container runs bounded on bridge', async () => {
    // Given - a freshly-upped docker-pilot world (worlds/default.yaml declares spwn:unix + spwn:git)
    await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

    // Then - the container is running with the declared dependency binaries present
    expect(result.exitCode).toBe(0);
    const neo = result.container('neo');
    expect(neo.running).toBe(true);

    const bash = await neo.exec('which bash');
    expect(bash.exitCode).toBe(0);
    expect(bash.stdout).toContain('bash');

    const git = await neo.exec('which git');
    expect(git.exitCode).toBe(0);
    expect(git.stdout).toContain('git');

    // Faculties (the tool list) is inlined into CLAUDE.md (scalpel: probing inlined markers, not a full golden)
    const claude = neo.file('/agents/neo/CLAUDE.md').content;
    expect(claude).toMatch(/spwn:unix/);
    expect(claude).toMatch(/git/);

    // And the container runs on the bridge network with a bounded pids limit
    const inspectData = neo.inspect.value as {
        HostConfig?: { NetworkMode?: string; PidsLimit?: number };
    };
    expect(inspectData.HostConfig?.NetworkMode).toBe('bridge');
    const pidsLimit = inspectData.HostConfig?.PidsLimit ?? 0;
    expect(pidsLimit).toBeGreaterThan(0);
});
