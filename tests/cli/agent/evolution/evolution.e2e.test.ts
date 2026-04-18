import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Agent evolution: dream, sleep, fork.
 *
 * These specs run against the `single-agent` fixture so the agent
 * lives inside the project under `spwn/agents/neo/`. Journal overlays
 * are copied in via the `agent/` seed handler; the CLI reads them
 * straight from disk when `agent dream`/`agent sleep` run.
 *
 * Mtime-sensitive behaviours (stale playbook/knowledge archival,
 * session pruning) are not covered here — the copying seed handler
 * does not preserve mtimes, so those assertions belong to the Go
 * unit tests that can backdate files directly.
 */

describe('spwn agent dream', () => {
    test('dream with no journal entries skips', async () => {
        const result = await spec('dream empty')
            .project('single-agent')
            .exec('agent dream neo')
            .run();

        expect(result.exitCode).toBe(0);
        const out = result.stderr.text;
        expect(out).toContain('Dreaming for agent "neo"');
        expect(out).toContain('no journal entries');
    });

    test('dream on a missing agent skips gracefully', async () => {
        const result = await spec('dream missing')
            .project('single-agent')
            .exec('agent dream ghost')
            .run();

        // Dream treats a missing journal the same as a missing agent —
        // It is a no-op, not an error. This mirrors the Go semantics.
        expect(result.exitCode).toBe(0);
        const out = result.stderr.text;
        expect(out).toContain('Dreaming for agent "ghost"');
        expect(out).toContain('no journal entries');
    });

    test('dream with journal entries writes auto-reflexion.md', async () => {
        const result = await spec('dream with journal')
            .project('single-agent')
            .seed('agent/neo/journal/2024-01-01.md')
            .seed('agent/neo/journal/2024-01-02.md')
            .exec('agent dream neo')
            .run();

        expect(result.exitCode).toBe(0);
        const reflexion = result.file('spwn/agents/neo/playbooks/auto-reflexion.md');
        expect(reflexion.exists).toBe(true);
        // The reflexion file must carry real content, not just an
        // Empty stub — dream should have written something useful.
        expect(reflexion.content.length).toBeGreaterThan(0);
        const out = result.stderr.text;
        expect(out).toMatch(/Entries analyzed\s+2/);
        expect(out).toMatch(/Success rate\s+\d+%/);
        expect(out).toMatch(/Completed\s+\d+/);
        expect(out).toMatch(/Failed\s+\d+/);
    });

    test('dream is idempotent — running twice leaves one reflexion', async () => {
        const result = await spec('dream twice')
            .project('single-agent')
            .seed('agent/neo/journal/2024-01-01.md')
            .exec(['agent dream neo', 'agent dream neo'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo/playbooks/auto-reflexion.md').exists).toBe(true);
        // Second dream still exits cleanly and the banner is present —
        // The deep content check is covered by the Go unit tests.
        expect(result.stderr.text).toContain('Entries analyzed');
    });
});

describe('spwn agent sleep', () => {
    test('sleep on a fresh agent reports zero-count archives', async () => {
        const result = await spec('sleep fresh')
            .project('single-agent')
            .exec('agent sleep neo')
            .run();

        expect(result.exitCode).toBe(0);
        const out = result.stderr.text;
        expect(out).toContain('Sleep cycle for agent "neo"');
        expect(out).toMatch(/Archived playbooks\s+0/);
        // Knowledge is world-scoped now, no longer in the sleep banner.
        expect(out).toMatch(/Pruned sessions\s+0/);
    });

    test('sleep on a missing agent is a no-op', async () => {
        const result = await spec('sleep missing')
            .project('single-agent')
            .exec('agent sleep ghost')
            .run();

        expect(result.exitCode).toBe(0);
        const out = result.stderr.text;
        expect(out).toContain('Sleep cycle for agent "ghost"');
        expect(out).toMatch(/Archived playbooks\s+0/);
        expect(out).toMatch(/Pruned sessions\s+0/);
    });

    test('sleep right after dream preserves the fresh reflexion', async () => {
        const result = await spec('dream then sleep')
            .project('single-agent')
            .seed('agent/neo/journal/2024-01-01.md')
            .exec(['agent dream neo', 'agent sleep neo'])
            .run();

        expect(result.exitCode).toBe(0);
        // Auto-reflexion was just written and is therefore fresh — it
        // Must survive the subsequent sleep cycle.
        expect(result.file('spwn/agents/neo/playbooks/auto-reflexion.md').exists).toBe(true);
    });
});

describe('spwn agent fork', () => {
    test('fork copies an agent to a new name', async () => {
        const result = await spec('fork neo')
            .project('single-agent')
            .exec('agent fork neo neo-v2')
            .run();

        expect(result.exitCode).toBe(0);
        const out = result.stderr.text;
        expect(out).toContain('Forking "neo" -> "neo-v2"');
        // identity/ is gone; Mind layers are skills/playbooks/journal.
        // SOUL.md is copied alongside the layers but isn't listed here.
        expect(out).toMatch(/Layers copied\s+skills, playbooks, journal/);
        expect(result.file('spwn/agents/neo-v2/SOUL.md').exists).toBe(true);
    });

    test('forking onto an existing target fails cleanly', async () => {
        const result = await spec('fork duplicate')
            .project('single-agent')
            .exec(['agent fork neo neo-v2', 'agent fork neo neo-v2'])
            .run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('fork-duplicate-target.txt');
        expect(result.stderr.text).not.toContain('panic:');
    });

    test('fork is symmetric with create in spwn.yaml', async () => {
        // Given - an initialised project (which already scaffolds a neo agent)
        // When - we fork neo to morpheus and re-run check
        // Then - check is clean because the fork also added a world
        const result = await spec('fork adds world')
            .project('empty')
            .exec(['init', 'agent fork neo morpheus', 'check'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/morpheus/agent.yaml').exists).toBe(true);
        expect(result.file('spwn.yaml').content).toContain('morpheus');
    });

    test('dream reflexion has no empty-id session row', async () => {
        // Given - a fresh agent whose journal may contain stub files
        // When - we run dream
        // Then - auto-reflexion never lists "- : <status>" rows
        const result = await spec('dream no phantom')
            .project('empty')
            .exec(['init', 'agent dream neo'])
            .run();

        expect(result.exitCode).toBe(0);
        const reflexion = result.file('spwn/agents/neo/playbooks/auto-reflexion.md');
        // Dream may skip when there are no entries; if the file does
        // Get written, every row must carry a real session id.
        if (reflexion.exists) {
            expect(reflexion.content).not.toMatch(/^- :/m);
        }
    });

    test('forked agent is inspectable via show', async () => {
        const result = await spec('fork then show')
            .project('single-agent')
            .exec(['agent fork neo neo-clone', 'agent show neo-clone'])
            .run();

        expect(result.exitCode).toBe(0);
        const out = result.stderr.text;
        expect(out).toMatch(/Agent:\s+neo-clone/);
        // Mind tree now renders skills/playbooks/journal; identity
        // collapsed into SOUL.md at the agent root.
        expect(out).toMatch(/skills\//);
    });
});
