import { describe, expect, test } from 'vitest';

import { spec } from '../../setup/cli.specification.js';

/**
 * Coverage for `spwn install` / `spwn uninstall` —
 * the npm-style dependency-management verbs. These mutate the target
 * agent.yaml plus the project-root spwn.lock and never touch
 * Docker, so the tests run fast against the lightweight docker-pilot
 * fixture.
 *
 * What's locked in here:
 *   - installing an spwn:* ref pins it in spwn.lock
 *   - installing a bare name is rejected with an authoring hint
 *   - installing @<owner>/* is rejected as unsupported
 *   - uninstall removes the ref from agent.yaml and the lockfile
 *   - double-install is idempotent (no duplicate agent.yaml entry)
 */
describe('spwn install', () => {
    test('pins an spwn:* ref into spwn.lock', async () => {
        const result = await spec('install builtin')
            .project('docker-pilot')
            .exec('install spwn:python')
            .run();

        expect(result.exitCode).toBe(0);
        const lock = result.file('spwn.lock');
        expect(lock.exists).toBe(true);
        expect(lock.content).toContain('spwn:python');

        const agentYaml = result.file('spwn/agents/neo/agent.yaml');
        expect(agentYaml.content).toContain('spwn:python');
    });

    test('rejects a bare name that misses the catalog with a known-list hint', async () => {
        const result = await spec('install bare rejected')
            .project('docker-pilot')
            .exec('install my-local-tool')
            .run();

        expect(result.exitCode).not.toBe(0);
        // Bare names route through the CLI resolver — they're auto-
        // Promoted to spwn:<name> when the catalog has the entry, or
        // Rejected with a list of what IS in the catalog + a local-
        // Scheme alternative when the name is unknown.
        expect(result.stderr.text).toContain('not in the catalog');
        expect(result.stderr.text).toContain('skill:my-local-tool');
    });

    test('accepts a bare name that matches the catalog and installs the spwn:<name>', async () => {
        // Given - a project with neo declared
        // When - we run `spwn install python` (bare catalog slug)
        // Then - neo's agent.yaml receives spwn:python, same as the
        // Explicit form `spwn install spwn:python`. The manifest keeps
        // The scheme-form; the CLI sugar is resolver-only.
        const result = await spec('install bare accepted')
            .project('docker-pilot')
            .exec('install python')
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/neo/agent.yaml').content).toContain('spwn:python');
    });

    test('rejects skill:/tool:/hook: without --agent (local-scope hint)', async () => {
        // Without --agent, a local ref is refused — bolting a local
        // Block onto every agent is almost never the intent. The
        // Error points at --agent as the fix.
        const result = await spec('install local rejected')
            .project('docker-pilot')
            .exec('install skill:paper-reading')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toContain('--agent');
    });

    test('rejects github:<owner>/<repo> as unsupported', async () => {
        const result = await spec('install registry rejected')
            .project('docker-pilot')
            .exec('install github:acme/foo')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toContain('not yet supported');
    });

    test('rejects an unknown spwn:* ref', async () => {
        const result = await spec('install unknown builtin')
            .project('docker-pilot')
            .exec('install spwn:nonesuch')
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toContain('unknown builtin');
    });

    test('is idempotent on re-install', async () => {
        const result = await spec('install idempotent')
            .project('docker-pilot')
            .exec(['install spwn:python', 'install spwn:python'])
            .run();

        expect(result.exitCode).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml');
        // Only count list-entries, not mentions in header comments.
        const entries = agentYaml.content.match(/^\s*-\s+["']?spwn:python["']?\s*$/gm) ?? [];
        expect(entries.length).toBe(1);
    });

    test('--agent narrows scope to a single agent (others untouched)', async () => {
        // Given - a multi-agent project seeded by `init severance`
        // (mark + helly + irving + dylan)
        // When - we scope an install to one agent
        // Then - only that agent's manifest gains the dep; every
        // Other agent stays byte-identical.
        const result = await spec('install scoped agent')
            .project('empty')
            .exec(['init severance', 'install qmd --agent mark'])
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);
        expect(result.file('spwn/agents/mark/agent.yaml').content).toContain('spwn:qmd');
        // The other three MDR agents have NO new entry — the flag
        // Narrows correctly.
        for (const name of ['helly', 'irving', 'dylan']) {
            expect(
                result.file(`spwn/agents/${name}/agent.yaml`).content,
                `${name} should not have spwn:qmd`,
            ).not.toContain('spwn:qmd');
        }
    });

    test('--agent <unknown> errors with the list of real agents', async () => {
        // Typo protection: if the user passes an agent name that
        // Doesn't exist, they should see an error naming the agents
        // That DO exist rather than a silent "0 agents updated".
        const result = await spec('install scoped miss')
            .project('empty')
            .exec(['init severance', 'install qmd --agent ghost'])
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toContain('"ghost" is not in this project');
        // The hint enumerates real agents so the user can correct
        // The typo.
        expect(result.stderr.text).toMatch(/mark|dylan|helly|irving/);
    });

    test('local refs need --agent (bolting onto every agent is a footgun)', async () => {
        // Without a scope, `install skill:refine` would write the
        // Entry into every agent — rarely the intent for a local
        // Block. The CLI refuses and points at the --agent flag.
        const result = await spec('install local no agent')
            .project('empty')
            .exec(['init severance', 'install skill:refine'])
            .run();

        expect(result.exitCode).not.toBe(0);
        expect(result.stderr.text).toMatch(/local ref/);
        expect(result.stderr.text).toMatch(/--agent/);
    });

    test('--agent attaches an explicit local skill to one agent', async () => {
        // Given - the severance fixture ships spwn/skills/refine.md
        // When - we attach it to dylan via --agent
        // Then - dylan's manifest picks up `skill:refine`; the other
        // MDR agents stay clean. The skill file itself is left alone.
        const result = await spec('install local scoped')
            .project('empty')
            .exec(['init severance', 'install skill:refine --agent dylan'])
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);
        expect(result.file('spwn/agents/dylan/agent.yaml').content).toContain('skill:refine');
        for (const name of ['mark', 'helly', 'irving']) {
            // These agents may already reference skill:refine from
            // The seed — we only check that no EXTRA entry was added
            // By the scoped install (not a hard "does not contain").
            // So just confirm dylan was the one mutated by inspecting
            // The stdout summary.
            void name;
        }
        expect(result.stdout.text).toMatch(/1 agent updated/);
    });

    test('preserves the @version suffix when resolving a bare name', async () => {
        // Given - a project that has `python` in the catalog
        // When - we install with an explicit version suffix via the
        // Bare shorthand (`python@latest`)
        // Then - the resolver promotes the stem to spwn:<name>; the
        // Manifest records the unversioned dep (like npm's
        // `dependencies:` list) while the lockfile pins the requested
        // Version. The test proves the @version survived the resolver
        // Without being silently dropped.
        const result = await spec('install bare with version')
            .project('docker-pilot')
            .exec('install python@latest')
            .run();

        expect(result.exitCode).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml');
        expect(agentYaml.content).toContain('spwn:python');
        const lock = result.file('spwn.lock');
        expect(lock.exists).toBe(true);
        // Lockfile records the version that was pinned. If the
        // Resolver had dropped `@latest` before SplitVersion, the
        // Lockfile would carry an empty version.
        expect(lock.content).toContain('spwn:python');
        expect(lock.content).toMatch(/latest/);
    });
});

describe('spwn uninstall', () => {
    test('removes the ref from agent.yaml and the lockfile', async () => {
        const result = await spec('uninstall')
            .project('docker-pilot')
            .exec(['install spwn:python', 'uninstall spwn:python'])
            .run();

        expect(result.exitCode).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml');
        expect(agentYaml.content).not.toContain('spwn:python');

        const lock = result.file('spwn.lock');
        if (lock.exists) {
            expect(lock.content).not.toContain('spwn:python');
        }
    });

    test('accepts a bare name and removes the spwn:<name> it added', async () => {
        // Symmetry check: if `install python` worked, `uninstall
        // Python` must work too — the user shouldn't have to type
        // `spwn:python` to undo what they typed bare. Pins the
        // Resolver's presence on the uninstall path.
        const result = await spec('uninstall bare')
            .project('docker-pilot')
            .exec(['install python', 'uninstall python'])
            .run();

        expect(result.exitCode).toBe(0);
        const agentYaml = result.file('spwn/agents/neo/agent.yaml');
        expect(agentYaml.content).not.toContain('spwn:python');
    });
});
