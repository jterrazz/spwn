import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * World physics. The v8 file shared one container across four read-only
 * assertions via beforeAll/afterAll (a nude `let world` — rule B5). Those
 * assertions are consolidated into a single cohesive test so the file
 * still boots exactly one container, now owned by `await using`.
 *
 * Legacy coverage dropped (features gone): `world inspect` no longer
 * surfaces a Constants block, and physics.md no longer materialises
 * CPU/Memory/Timeout constants.
 */

test('up inlines physics and faculties into CLAUDE.md and runs on the bridge network', async () => {
    // Given - a freshly-upped docker-pilot world
    await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

    // Then - the container is running with world context inlined into the agent prompt
    expect(result.exitCode).toBe(0);
    const neo = result.container('neo');
    expect(neo.running).toBe(true);

    /*
     * Physics and faculties are inlined into every agent's CLAUDE.md
     * (previously separate /world/*.md files) so the system prompt is
     * self-contained at boot. Scalpel: probing inlined section markers,
     * not a stable full golden.
     */
    const claude = neo.file('/agents/neo/CLAUDE.md').content;
    expect(claude).toMatch(/## Physics/);
    expect(claude).toMatch(/## Faculties/);
    expect(claude).toMatch(/network/i);
    expect(claude).toMatch(/Laws/);
    expect(claude).toMatch(/\/workspace/);
    expect(claude).toMatch(/spwn:unix/);

    // And the default network mode is bridge so agents can reach the host
    const inspectData = neo.inspect.value as {
        HostConfig?: { NetworkMode?: string };
    };
    expect(inspectData.HostConfig?.NetworkMode).toBe('bridge');
});
