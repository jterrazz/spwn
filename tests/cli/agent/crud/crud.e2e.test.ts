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
        expect(result.file('spwn-home/agents/neo/core/profile.md').exists).toBe(true);
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

        expect(result.exitCode).not.toBe(0);
        result.stderr.toContain('agent "neo" already exists');
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
        expect(result.stderr.text).toMatch(/core\/\s+profile\.md/);
        expect(result.stderr.text).toMatch(/skills\/\s+\(empty\)/);
        expect(result.stderr.text).toMatch(/knowledge\/\s+\(empty\)/);
        expect(result.stderr.text).toMatch(/playbooks\/\s+\(empty\)/);
        expect(result.stderr.text).toMatch(/journal\/\s+\(empty\)/);
    });

    test('show on a missing agent errors cleanly', async () => {
        const result = await isolated('show missing').exec('agent show ghost').run();

        expect(result.exitCode).not.toBe(0);
        result.stderr.toContain('agent "ghost" not found');
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

        expect(result.exitCode).not.toBe(0);
        result.stderr.toContain('agent "ghost" not found');
        expect(result.stderr.text).not.toContain('panic:');
    });

    test('rm then show reports the agent as not found', async () => {
        const result = await isolated('rm then show')
            .exec(['agent create temp', 'agent rm temp', 'agent show temp'])
            .run();

        expect(result.exitCode).not.toBe(0);
        result.stderr.toContain('agent "temp" not found');
    });

    test('talk without a world fails with a helpful error', async () => {
        const result = await isolated('talk no world')
            .exec(['agent create neo', 'agent talk neo hello'])
            .run();

        expect(result.exitCode).not.toBe(0);
        result.stderr.toContain('agent "neo" is not in any active world');
    });
});
