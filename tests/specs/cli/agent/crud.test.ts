import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Agent CRUD surface.
 *
 * Each spec gets a fresh temp project with an isolated
 * `$WORKDIR/spwn-home` SPWN_HOME, so agent state never leaks between
 * specs. `agent create` writes into `$SPWN_HOME/agents/<name>/` — the
 * empty fixture has no `spwn.yaml`, which keeps worlds out of the
 * picture and makes `agent ls --json` stable. The runner is
 * docker-aware, so every result binds with `await using` (rule B5).
 */

// Isolated global-mode home so agent state never leaks between specs.
const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('agent crud', () => {
    test('create writes the mind structure to disk', async () => {
        // Given - an isolated global home
        await using result = await isolated().exec('agent create neo');

        // Then - the on-disk Mind exists with its Soul plus the two layer dirs
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.file('spwn-home/agents/neo/playbooks').exists).toBe(true);
        expect(result.file('spwn-home/agents/neo/journal').exists).toBe(true);
        expect(result.file('spwn-home/agents/neo/skills').exists).toBe(false);
        expect(result.file('spwn-home/agents/neo/knowledge').exists).toBe(false);
        // The creation banner is path-free and deterministic (byte-for-byte golden)
        expect(result.stderr).toMatch('create-neo.txt');
    });

    test('creating the same agent twice fails cleanly', async () => {
        // Given - the same agent created twice in one chain
        await using result = await isolated().exec(['agent create neo', 'agent create neo']);

        // Then - exit 1 with the canonical duplicate banner and no panic
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('create-duplicate.txt');
        expect(result.stderr).not.toContain('panic:');
    });

    test('ls --json lists created agents structurally', async () => {
        // Given - two agents created then listed as JSON
        await using result = await isolated().exec([
            'agent create neo',
            'agent create trinity',
            'agent ls --json',
        ]);

        // Then - the JSON envelope lists both agents unattached
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch('ls-two-agents.json');
    });

    test('ls --json on an empty home returns no agents', async () => {
        // Given - an isolated home with no agents created
        await using result = await isolated().exec('agent ls --json');

        // Then - the JSON envelope reports an empty roster
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch('ls-empty.json');
    });

    test('show prints agent details with all mind layers', async () => {
        // Given - a created agent inspected via show
        await using result = await isolated().exec(['agent create neo', 'agent show neo']);

        // Then - the Mind tree renders on stderr (scalpel: regex over the ANSI mind-tree render)
        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/Agent:\s+neo/);
        expect(stderr).toMatch(/playbooks\/\s+\(empty\)/);
        expect(stderr).toMatch(/journal\/\s+\(empty\)/);
    });

    test('show on a missing agent errors cleanly', async () => {
        // Given - show against an agent that was never created
        await using result = await isolated().exec('agent show ghost');

        // Then - exit 1 with the canonical not-found banner and no panic
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('show-missing.txt');
        expect(result.stderr).not.toContain('panic:');
    });

    test('rm deletes the agent from disk', async () => {
        // Given - a throwaway agent created then removed
        await using result = await isolated().exec(['agent create temp', 'agent rm temp']);

        // Then - the on-disk dir is gone (scalpel: banner presence probe)
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/agents/temp').exists).toBe(false);
        expect(result.stderr).toContain('Deleted agent');
    });

    test('rm on a missing agent errors cleanly', async () => {
        // Given - rm against an agent that was never created
        await using result = await isolated().exec('agent rm ghost');

        // Then - exit 1 with the canonical delete-failed banner and no panic
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('rm-missing.txt');
        expect(result.stderr).not.toContain('panic:');
    });

    test('rm then show reports the agent as not found', async () => {
        // Given - an agent created, removed, then shown in one chain
        await using result = await isolated().exec([
            'agent create temp',
            'agent rm temp',
            'agent show temp',
        ]);

        // Then - exit 1 with the not-found error banner
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('rm-then-show-not-found.txt');
    });

    test('agent new is an alias for agent create', async () => {
        // Given - create via both the canonical verb and the alias
        await using created = await isolated().exec('agent create neo');
        await using aliased = await isolated().exec('agent new neo');

        // Then - both produce the same on-disk Mind layout
        expect(created.exitCode).toBe(0);
        expect(aliased.exitCode).toBe(0);
        for (const path of [
            'spwn-home/agents/neo/SOUL.md',
            'spwn-home/agents/neo/playbooks',
            'spwn-home/agents/neo/journal',
        ]) {
            expect(created.file(path).exists).toBe(true);
            expect(aliased.file(path).exists).toBe(true);
        }
    });

    test('agent create --force succeeds even when the agent already exists', async () => {
        // Given - an agent already scaffolded, then re-created with --force
        await using result = await isolated().exec([
            'agent create neo',
            'agent create neo --force',
        ]);

        // Then - no error and the Mind layers are still on disk
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/agents/neo/SOUL.md').exists).toBe(true);
    });

    test('agent create without --force still rejects duplicates', async () => {
        // Given - the agent already scaffolded, then re-created without --force
        await using result = await isolated().exec(['agent create neo', 'agent create neo']);

        // Then - exit 1 (scalpel: message-substring probe; the full banner is golden-checked in the duplicate test above)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toContain('already exists');
    });

    test('agent create rejects invalid names', async () => {
        // Given - a name with a space (would corrupt spwn.yaml downstream)
        await using result = await isolated().exec('agent create "bad name"');

        // Then - exit 1, nothing written to disk (scalpel: message-substring probe)
        expect(result.exitCode).toBe(1);
        expect(result.file('spwn-home/agents/bad name').exists).toBe(false);
        const stderr = result.stderr.text;
        expect(stderr).toMatch(/invalid/i);
    });

    test('agent create inside a project scaffolds a check-valid agent', async () => {
        // Given - an initialised empty project with a fresh agent, re-checked
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init', 'agent new trinity', 'check']);

        // Then - check passes and the full mind tree is on disk
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/trinity/agent.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/trinity/AGENTS.md').exists).toBe(true);
        expect(result.file('spwn/agents/trinity/SOUL.md').exists).toBe(true);
    });

    test('agent rm cleans the manifest so check stays green', async () => {
        // Given - a project with an auto-registered agent, then removed and re-checked
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init', 'agent new trinity', 'agent rm trinity', 'check']);

        // Then - check passes and spwn.yaml no longer mentions the agent
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn.yaml').content).not.toContain('trinity');
    });

    test('agent inspect does not duplicate journal as sessions', async () => {
        // Given - an initialised project (init scaffolds neo) inspected
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init', 'agent inspect neo']);

        // Then - the legacy Sessions block is suppressed (scalpel: absence probe on the tree render)
        expect(result.exitCode).toBe(0);
        expect(result.stderr.text).not.toMatch(/Sessions:/);
    });

    test('talk without a world fails with a helpful error', async () => {
        // Given - an orphan agent with no active world, talked to
        await using result = await isolated().exec(['agent create neo', 'agent talk neo hello']);

        // Then - exit 1 with the canonical no-active-world error banner
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch('talk-no-active-world.txt');
    });
});
