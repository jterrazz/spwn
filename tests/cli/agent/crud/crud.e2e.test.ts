import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Agent CRUD surface.
 *
 * Each spec gets a fresh temp project with an isolated
 * `$WORKDIR/spwn-home` SPWN_HOME, so agent state never leaks between
 * specs. `agent create` writes into `$SPWN_HOME/agents/<name>/` — the
 * empty fixture has no `spwn.yaml`, which keeps worlds out of the
 * picture and makes `agent ls --json` stable.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('spwn agent CRUD', () => {
    test('create writes the 5-layer Mind to disk', async () => {
        const result = await isolated('create neo').exec('agent create neo').run();

        expect(result.exitCode).toBe(0);
        // Structural: the on-disk Mind exists with a profile.
        expect(result.file('spwn-home/agents/neo/identity/profile.md').exists).toBe(true);
        expect(result.file('spwn-home/agents/neo/skills').exists).toBe(true);
        expect(result.file('spwn-home/agents/neo/knowledge').exists).toBe(true);
        expect(result.file('spwn-home/agents/neo/playbooks').exists).toBe(true);
        expect(result.file('spwn-home/agents/neo/journal').exists).toBe(true);
        // Smoke-check the status banner so regressions in the CLI UX
        // Are caught without pinning the full text.
        result.stderr.toContain('Created agent');
    });

    test('creating the same agent twice fails cleanly', async () => {
        const result = await isolated('create duplicate')
            .exec(['agent create neo', 'agent create neo'])
            .run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('create-duplicate.txt');
        expect(result.stderr.text).not.toContain('panic:');
    });

    test('ls --json lists created agents structurally', async () => {
        const result = await isolated('ls json')
            .exec(['agent create neo', 'agent create trinity', 'agent ls --json'])
            .run();

        expect(result.exitCode).toBe(0);
        await result.json.toMatch('ls-two-agents.json');
    });

    test('ls --json on an empty home returns no agents', async () => {
        const result = await isolated('ls empty json').exec('agent ls --json').run();

        expect(result.exitCode).toBe(0);
        await result.json.toMatch('ls-empty.json');
    });

    test('show prints agent details with all Mind layers', async () => {
        const result = await isolated('show neo')
            .exec(['agent create neo', 'agent show neo'])
            .run();

        expect(result.exitCode).toBe(0);
        // `agent show` renders the Mind tree on stderr in spwn's UX.
        expect(result.stderr.text).toMatch(/Agent:\s+neo/);
        expect(result.stderr.text).toMatch(/identity\/\s+profile\.md/);
        expect(result.stderr.text).toMatch(/skills\/\s+\(empty\)/);
        expect(result.stderr.text).toMatch(/knowledge\/\s+\(empty\)/);
        expect(result.stderr.text).toMatch(/playbooks\/\s+\(empty\)/);
        expect(result.stderr.text).toMatch(/journal\/\s+\(empty\)/);
    });

    test('show on a missing agent errors cleanly', async () => {
        const result = await isolated('show missing').exec('agent show ghost').run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('show-missing.txt');
        expect(result.stderr.text).not.toContain('panic:');
    });

    test('rm deletes the agent from disk', async () => {
        const result = await isolated('rm neo').exec(['agent create temp', 'agent rm temp']).run();

        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/agents/temp').exists).toBe(false);
        result.stderr.toContain('Deleted agent');
    });

    test('rm on a missing agent errors cleanly', async () => {
        const result = await isolated('rm missing').exec('agent rm ghost').run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('rm-missing.txt');
        expect(result.stderr.text).not.toContain('panic:');
    });

    test('rm then show reports the agent as not found', async () => {
        const result = await isolated('rm then show')
            .exec(['agent create temp', 'agent rm temp', 'agent show temp'])
            .run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('rm-then-show-not-found.txt');
    });

    test('agent new is an alias for agent create', async () => {
        // Given - isolated home
        // When - create via both the canonical verb and the alias
        // Then - both produce the same on-disk Mind layout
        const created = await isolated('create via create').exec('agent create neo').run();
        const aliased = await isolated('create via new').exec('agent new neo').run();

        expect(created.exitCode).toBe(0);
        expect(aliased.exitCode).toBe(0);

        for (const path of [
            'spwn-home/agents/neo/identity/profile.md',
            'spwn-home/agents/neo/skills',
            'spwn-home/agents/neo/knowledge',
            'spwn-home/agents/neo/playbooks',
            'spwn-home/agents/neo/journal',
        ]) {
            expect(created.file(path).exists).toBe(true);
            expect(aliased.file(path).exists).toBe(true);
        }
    });

    test('agent create --force succeeds even when the agent already exists', async () => {
        // Given - an agent already scaffolded
        // When - re-running create with --force
        // Then - no "already exists" error; Mind layers still on disk
        const result = await isolated('create force')
            .exec(['agent create neo', 'agent create neo --force'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('spwn-home/agents/neo/identity/profile.md').exists).toBe(true);
    });

    test('agent create without --force still rejects duplicates', async () => {
        // Given - the agent already scaffolded
        // When - re-running without --force
        // Then - exit 1 with the "already exists" error path
        const result = await isolated('create dup no force')
            .exec(['agent create neo', 'agent create neo'])
            .run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toMatch(/already exists/i);
    });

    test('agent create rejects invalid names', async () => {
        // Given - a name with a space (would corrupt spwn.yaml downstream)
        // When - running agent create
        // Then - the command exits 1 and writes nothing to disk
        const result = await isolated('create bad name').exec('agent create "bad name"').run();

        expect(result.exitCode).toBe(1);
        expect(result.file('spwn-home/agents/bad name').exists).toBe(false);
        expect(result.stderr.text).toMatch(/invalid/i);
    });

    test('agent remove --pack rejects packs that were never attached', async () => {
        // Given - neo with no packages
        // When - remove --pack for a ref that isn't in its composition
        // Then - exit 1 with a "nothing to remove" message, no green check
        const result = await isolated('remove absent package')
            .exec(['agent create neo', 'agent remove neo --pack @spwn/never-added'])
            .run();

        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toMatch(/not attached|nothing to remove/i);
        expect(result.stderr.text).not.toMatch(/Composition updated/);
    });

    test('agent create inside a project scaffolds a check-valid agent', async () => {
        // Given - an initialised empty project
        // When - we create an agent and re-run check
        // Then - check passes and the full mind tree is on disk
        const result = await spec('create inside project')
            .project('empty')
            .exec(['init', 'agent new trinity', 'check'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/trinity/agent.yaml').exists).toBe(true);
        expect(result.file('spwn/agents/trinity/AGENTS.md').exists).toBe(true);
        expect(result.file('spwn/agents/trinity/identity').exists).toBe(true);
    });

    test('agent rm cleans the manifest so check stays green', async () => {
        // Given - a project with one agent auto-registered in its world
        // When - we rm the agent and re-run check
        // Then - check passes (the world reference is gone)
        const result = await spec('rm cleans manifest')
            .project('empty')
            .exec(['init', 'agent new trinity', 'agent rm trinity', 'check'])
            .run();

        expect(result.exitCode).toBe(0);
        // And spwn.yaml no longer mentions trinity
        expect(result.file('spwn.yaml').content).not.toContain('trinity');
    });

    test('agent add rejects unknown package refs', async () => {
        // Given - an initialised project (init scaffolds neo)
        // When - we try to add a package that is not in the catalog
        // Then - exit 1 and agent.yaml is not corrupted
        const result = await spec('add bogus package')
            .project('empty')
            .exec(['init', 'agent add neo --pack @spwn/nonexistent'])
            .run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('add-unknown-package.txt');
    });

    test('agent inspect does not duplicate journal as Sessions', async () => {
        // Given - an initialised project (init scaffolds neo)
        // When - running agent inspect
        // Then - the legacy Sessions: block is suppressed
        const result = await spec('inspect no dup')
            .project('empty')
            .exec(['init', 'agent inspect neo'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.stderr.text).not.toMatch(/Sessions:/);
    });

    test('talk without a world fails with a helpful error', async () => {
        const result = await isolated('talk no world')
            .exec(['agent create neo', 'agent talk neo hello'])
            .run();

        expect(result.exitCode).toBe(1);
        await result.stderr.toMatch('talk-no-active-world.txt');
    });
});
