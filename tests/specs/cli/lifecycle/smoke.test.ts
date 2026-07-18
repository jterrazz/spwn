import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Non-docker agent lifecycle: init -> agent ls -> dream -> rm, plus the
 * scaffold layout and multi-agent coexistence. The container-backed spawn flow
 * lives in the docker-gated lifecycle aspects; this file covers everything a
 * user can do without a container. Chained flows carry dynamic init banners, so
 * output probes stay scalpels; on-disk shape is asserted directly. Every result
 * binds with `await using` (rule B5).
 */

describe('agent lifecycle (cli-only)', () => {
    test('init -> agent ls -> dream -> rm round-trip', async () => {
        // Given - an empty dir taken through the full non-docker journey in one chain
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec(['init --name lifecycle', 'agent ls', 'agent dream neo', 'agent rm neo']);

        // Then - the final rm prints its delete banner and the agent dir is gone (scalpel: chained tail)
        expect(result.exitCode).toBe(0);
        expect(result.stderr.text).toMatch(/Deleted agent\s+neo/);
        expect(result.file('spwn/agents/neo').exists).toBe(false);
    });

    test('init scaffolds the default agent with the expected mind layout', async () => {
        // Given - an empty dir scaffolded with the starter project
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init --name layout-check');

        // Then - SOUL.md plus the playbooks/journal layers land under spwn/agents/neo, knowledge under spwn/
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo').exists).toBe(true);
        expect(result.file('spwn/agents/neo/identity').exists).toBe(false);
        expect(result.file('spwn/agents/neo/skills').exists).toBe(false);
        expect(result.file('spwn/agents/neo/playbooks').exists).toBe(true);
        expect(result.file('spwn/agents/neo/journal').exists).toBe(true);
        expect(result.file('spwn/agents/neo/knowledge').exists).toBe(false);
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.file('spwn/agents/neo/agent.yaml').exists).toBe(true);
        expect(result.file('spwn/knowledge').exists).toBe(true);
        expect(result.file('knowledge').exists).toBe(false);
        expect(result.file('spwn/worlds').exists).toBe(false);
    });

    test('multiple agents can coexist in the same project', async () => {
        // Given - a scaffolded project with a second agent created then listed
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init --name multi-agent', 'agent create trinity', 'agent ls']);

        // Then - both minds are on disk and both names render in the listing (scalpel: chained tail)
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo').exists).toBe(true);
        expect(result.file('spwn/agents/trinity').exists).toBe(true);
        expect(result.stderr).toContain('neo');
        expect(result.stderr).toContain('trinity');
    });
});
