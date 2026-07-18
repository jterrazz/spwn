import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn install` / `spwn uninstall` — the npm-style dependency verbs. They
 * mutate the target agent.yaml plus the project-root spwn.lock and never touch
 * Docker, so the specs run fast against the lightweight docker-pilot fixture.
 * The runner is docker-aware, so every result binds with `await using` even
 * though nothing spawns a container (rule B5). Error-wording checks stay
 * scalpels (rule D11(e)): cobra/resolver phrasing is third-party-shaped.
 */

describe('spwn install', () => {
    test('pins an spwn:* ref into spwn.lock', async () => {
        // Given - a bare docker-pilot project
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('install spwn:python');

        // Then - the lockfile and neo's manifest both record the canonical ref
        expect(result.exitCode).toBe(0);
        const lock = result.file('spwn.lock');
        expect(lock.exists).toBe(true);
        expect(lock.content).toContain('spwn:python');
        expect(result.file('spwn/agents/neo/agent.yaml').content).toContain('spwn:python');
    });

    test('rejects a bare name that misses the catalog with a known-list hint', async () => {
        // Given - a bare name with no catalog entry
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('install my-local-tool');

        // Then - rejected with the catalog hint and a local-scheme alternative (scalpel: resolver wording)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('not in the catalog');
        expect(result.stderr).toContain('skill/my-local-tool');
    });

    test('accepts a bare name that matches the catalog and installs the spwn:<name>', async () => {
        // Given - "python" is a catalog slug; the resolver promotes it before disk
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('install python');

        // Then - the manifest keeps the scheme form, same as the explicit install
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo/agent.yaml').content).toContain('spwn:python');
    });

    test('rejects skill/, tool/, hook/ refs without --agent', async () => {
        // Given - a local ref with no scope; bolting it onto every agent is rarely intended
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('install skill/paper-reading');

        // Then - refused, pointing at --agent as the fix (scalpel: error wording)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('--agent');
    });

    test('rejects github:<owner>/<repo> as unsupported', async () => {
        // Given - a remote registry ref
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('install github:acme/foo');

        // Then - refused as not yet supported (scalpel: error wording)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('not yet supported');
    });

    test('rejects an unknown spwn:* ref', async () => {
        // Given - an spwn:* ref with no matching built-in
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('install spwn:nonesuch');

        // Then - refused as an unknown builtin (scalpel: error wording)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('unknown builtin');
    });

    test('is idempotent on re-install', async () => {
        // Given - the same ref installed twice in one chain
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['install spwn:python', 'install spwn:python']);

        // Then - the manifest carries exactly one list entry (regex: count list entries, not comment mentions)
        expect(result.exitCode).toBe(0);
        const entries =
            result
                .file('spwn/agents/neo/agent.yaml')
                .content.match(/^\s*-\s+["']?spwn:python["']?\s*$/gm) ?? [];
        expect(entries.length).toBe(1);
    });

    test('errors when the project has no agents declared', async () => {
        // Given - a project whose only agent has been removed, leaving no consumers
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init', 'agent rm neo', 'install python']);

        // Then - refused with a pointer at `spwn agent new` (scalpel: error wording)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('no agents declared');
        expect(result.stderr).toContain('spwn agent new');
    });

    test('project-wide install reaches every agent in a multi-agent project', async () => {
        // Given - the severance gallery declares four agents
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init severance', 'install qmd']);

        // Then - every agent's manifest picks up the ref and the banner reports the fan-out
        expect(result.exitCode, result.stderr.text).toBe(0);
        for (const name of ['mark', 'helly', 'irving', 'dylan']) {
            expect(
                result.file(`spwn/agents/${name}/agent.yaml`).content,
                `${name} missing spwn:qmd`,
            ).toContain('spwn:qmd');
        }
        expect(result.stdout).toContain('4 agents updated');
    });

    test('project-wide install is idempotent across mixed pre-existing state', async () => {
        // Given - one agent already carries the ref (scoped), the other three do not
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init severance', 'install qmd --agent mark', 'install qmd']);

        // Then - every agent ends with exactly one entry; mark's prior entry is not duplicated
        expect(result.exitCode, result.stderr.text).toBe(0);
        for (const name of ['mark', 'helly', 'irving', 'dylan']) {
            const entries =
                result
                    .file(`spwn/agents/${name}/agent.yaml`)
                    .content.match(/^\s*-\s+["']?spwn:qmd["']?\s*$/gm) ?? [];
            expect(entries.length, `${name} should have exactly 1 spwn:qmd entry`).toBe(1);
        }
    });

    test('--agent narrows scope to a single agent, leaving others untouched', async () => {
        // Given - a four-agent severance project
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init severance', 'install qmd --agent mark']);

        // Then - only mark gains the dep; the other three stay clean
        expect(result.exitCode, result.stderr.text).toBe(0);
        expect(result.file('spwn/agents/mark/agent.yaml').content).toContain('spwn:qmd');
        for (const name of ['helly', 'irving', 'dylan']) {
            expect(
                result.file(`spwn/agents/${name}/agent.yaml`).content,
                `${name} should not have spwn:qmd`,
            ).not.toContain('spwn:qmd');
        }
    });

    test('--agent <unknown> errors with the list of real agents', async () => {
        // Given - a scope naming an agent that does not exist
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init severance', 'install qmd --agent ghost']);

        // Then - refused, enumerating the real agents so the typo can be fixed (scalpel: error wording)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('"ghost" is not in this project');
        expect(result.stderr.text).toMatch(/mark|dylan|helly|irving/);
    });

    test('local refs need --agent, since bolting onto every agent is a footgun', async () => {
        // Given - a local skill ref with no scope
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init severance', 'install skill/refine']);

        // Then - refused, pointing at --agent (scalpel: error wording)
        expect(result.exitCode).not.toBe(0);
        expect(result.stderr).toContain('local ref');
        expect(result.stderr).toContain('--agent');
    });

    test('--agent attaches an explicit local skill to one agent', async () => {
        // Given - the severance fixture ships spwn/skills/refine.md, attached to dylan via --agent
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init severance', 'install skill/refine --agent dylan']);

        // Then - dylan's manifest picks up the skill and the banner reports a single update
        expect(result.exitCode, result.stderr.text).toBe(0);
        expect(result.file('spwn/agents/dylan/agent.yaml').content).toContain('skill/refine');
        expect(result.stdout).toContain('1 agent updated');
    });

    test('preserves the @version suffix when resolving a bare name', async () => {
        // Given - a bare install with an explicit version suffix
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec('install python@latest');

        // Then - the manifest records the unversioned dep while the lockfile pins the requested version
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo/agent.yaml').content).toContain('spwn:python');
        const lock = result.file('spwn.lock');
        expect(lock.exists).toBe(true);
        expect(lock.content).toContain('spwn:python');
        expect(lock.content).toContain('latest');
    });
});

describe('spwn uninstall', () => {
    test('removes the ref from agent.yaml and the lockfile', async () => {
        // Given - a ref installed then immediately uninstalled
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['install spwn:python', 'uninstall spwn:python']);

        // Then - the manifest drops the ref and any lockfile pin is gone
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo/agent.yaml').content).not.toContain('spwn:python');
        const lock = result.file('spwn.lock');
        if (lock.exists) {
            expect(lock.content).not.toContain('spwn:python');
        }
    });

    test('scoped uninstall keeps the lockfile pin when others still carry the ref', async () => {
        // Given - all four agents carry qmd, then it is dropped from mark only
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init severance', 'install qmd', 'uninstall qmd --agent mark']);

        // Then - mark loses it, the others keep it, and the lockfile pin survives
        expect(result.exitCode, result.stderr.text).toBe(0);
        expect(result.file('spwn/agents/mark/agent.yaml').content).not.toContain('spwn:qmd');
        for (const name of ['helly', 'irving', 'dylan']) {
            expect(
                result.file(`spwn/agents/${name}/agent.yaml`).content,
                `${name} should still carry spwn:qmd`,
            ).toContain('spwn:qmd');
        }
        expect(result.file('spwn.lock').content).toContain('spwn:qmd');
    });

    test('scoped uninstall drops the lockfile pin when the last carrier loses it', async () => {
        // Given - only mark carries qmd, then it is dropped from mark
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .exec(['init severance', 'install qmd --agent mark', 'uninstall qmd --agent mark']);

        // Then - no agent carries the ref and the lockfile no longer pins it
        expect(result.exitCode, result.stderr.text).toBe(0);
        for (const name of ['mark', 'helly', 'irving', 'dylan']) {
            expect(
                result.file(`spwn/agents/${name}/agent.yaml`).content,
                `${name} should not carry spwn:qmd`,
            ).not.toContain('spwn:qmd');
        }
        const lock = result.file('spwn.lock');
        if (lock.exists) {
            expect(lock.content).not.toContain('spwn:qmd');
        }
    });

    test('accepts a bare name and removes the spwn:<name> it added', async () => {
        // Given - a ref installed and uninstalled both by its bare name
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(['install python', 'uninstall python']);

        // Then - the resolver is present on the uninstall path too; the manifest is clean
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo/agent.yaml').content).not.toContain('spwn:python');
    });
});
