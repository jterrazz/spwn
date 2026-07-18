import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Agent evolution: dream, sleep, fork.
 *
 * These specs run against the `single-agent` fixture so the agent lives
 * inside the project under `spwn/agents/neo/`. The `journal/` overlay
 * layers two journal entries onto neo; the CLI reads them straight from
 * disk when `agent dream`/`agent sleep` run.
 *
 * Mtime-sensitive behaviours (stale playbook/knowledge archival, session
 * pruning) are not covered here — fixture layering does not backdate
 * files, so those assertions belong to the Go unit tests. The runner is
 * docker-aware, so every result binds with `await using` (rule B5).
 */

describe('agent dream', () => {
    test('dream with no journal entries skips', async () => {
        // Given - a project agent with an empty journal
        await using result = await cli.fixture('$FIXTURES/single-agent/').exec('agent dream neo');

        // Then - dream is a clean no-op (scalpel: banner presence probe)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Dreaming for agent "neo"');
        expect(result.stderr).toContain('no journal entries');
    });

    test('dream on a missing agent skips gracefully', async () => {
        // Given - dream against an agent that does not exist
        await using result = await cli.fixture('$FIXTURES/single-agent/').exec('agent dream ghost');

        // Then - a missing journal is a no-op, not an error (scalpel: banner presence probe)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Dreaming for agent "ghost"');
        expect(result.stderr).toContain('no journal entries');
    });

    test('dream with journal entries writes auto-reflexion.md', async () => {
        // Given - two journal entries layered onto neo
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .fixture('journal/')
            .exec('agent dream neo');

        // Then - a non-empty reflexion is written and the banner counts both entries
        expect(result.exitCode).toBe(0);
        const reflexion = result.file('spwn/agents/neo/playbooks/auto-reflexion.md');
        expect(reflexion.exists).toBe(true);
        expect(reflexion.content.length).toBeGreaterThan(0);
        // Scalpel: regex over the dynamic dream summary banner
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/Entries analyzed\s+2/);
        expect(stderr).toMatch(/Success rate\s+\d+%/);
        expect(stderr).toMatch(/Completed\s+\d+/);
        expect(stderr).toMatch(/Failed\s+\d+/);
    });

    test('dream is idempotent — running twice leaves one reflexion', async () => {
        // Given - the journal overlay, dreamt over twice in one chain
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .fixture('journal/')
            .exec(['agent dream neo', 'agent dream neo']);

        // Then - the reflexion still exists and the second dream ran cleanly (scalpel: banner presence probe)
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo/playbooks/auto-reflexion.md').exists).toBe(true);
        expect(result.stderr).toContain('Entries analyzed');
    });
});

describe('agent sleep', () => {
    test('sleep on a fresh agent reports zero-count archives', async () => {
        // Given - a fresh project agent put to sleep
        await using result = await cli.fixture('$FIXTURES/single-agent/').exec('agent sleep neo');

        // Then - the sleep banner reports zero archives and zero pruned sessions (scalpel: regex over the banner)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Sleep cycle for agent "neo"');
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/Archived playbooks\s+0/);
        expect(stderr).toMatch(/Pruned sessions\s+0/);
    });

    test('sleep on a missing agent is a no-op', async () => {
        // Given - sleep against an agent that does not exist
        await using result = await cli.fixture('$FIXTURES/single-agent/').exec('agent sleep ghost');

        // Then - the sleep banner still reports zero counts (scalpel: regex over the banner)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Sleep cycle for agent "ghost"');
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/Archived playbooks\s+0/);
        expect(stderr).toMatch(/Pruned sessions\s+0/);
    });

    test('sleep right after dream preserves the fresh reflexion', async () => {
        // Given - a dream followed immediately by a sleep
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .fixture('journal/')
            .exec(['agent dream neo', 'agent sleep neo']);

        // Then - the freshly written reflexion survives the sleep cycle
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo/playbooks/auto-reflexion.md').exists).toBe(true);
    });
});

describe('agent fork', () => {
    test('fork copies an agent to a new name', async () => {
        // Given - neo forked to a new name
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .exec('agent fork neo neo-v2');

        // Then - the fork banner names the copied layers and the new Soul is on disk (scalpel: regex over the banner)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Forking "neo" -> "neo-v2"');
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/Layers copied\s+playbooks, journal/);
        expect(result.file('spwn/agents/neo-v2/SOUL.md').exists).toBe(true);
    });

    test('forking onto an existing target fails cleanly', async () => {
        // Given - the same fork attempted twice
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .exec(['agent fork neo neo-v2', 'agent fork neo neo-v2']);

        // Then - exit 1 with the canonical fork-failed banner and no panic
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('fork-duplicate-target.txt');
        expect(result.stderr).not.toContain('panic:');
    });

    test('fork is symmetric with create in spwn.yaml', async () => {
        // Given - an initialised project, forked to a new agent, re-checked
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init', 'agent fork neo morpheus', 'check']);

        // Then - check is clean because the fork also added a world
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/morpheus/agent.yaml').exists).toBe(true);
        expect(result.file('spwn.yaml').content).toContain('morpheus');
    });

    test('dream reflexion has no empty-id session row', async () => {
        // Given - a fresh agent dreamt with no real journal entries
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init', 'agent dream neo']);

        // Then - if a reflexion is written, no row carries an empty session id (scalpel: conditional regex probe)
        expect(result.exitCode).toBe(0);
        const reflexion = result.file('spwn/agents/neo/playbooks/auto-reflexion.md');
        if (reflexion.exists) {
            expect(reflexion.content).not.toMatch(/^- :/m);
        }
    });

    test('forked agent is inspectable via show', async () => {
        // Given - neo forked then the clone inspected via show
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .exec(['agent fork neo neo-clone', 'agent show neo-clone']);

        // Then - the Mind tree renders for the clone (scalpel: regex over the mind-tree render)
        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/Agent:\s+neo-clone/);
        expect(stderr).toMatch(/playbooks\//);
    });
});
