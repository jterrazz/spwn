import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Severance catalog full-cycle smoke test.
 *
 * The severance entry is the canonical 4-agent showcase: Mark (chief)
 * plus Helly/Irving/Dylan (workers), all in the `mdr` world. It
 * exercises:
 *
 *   - catalog install via `spwn init severance` (bare → spwn:severance)
 *   - project-root knowledge (./knowledge/ ships at project root)
 *   - four agent directories, each with a distinct SOUL.md
 *   - the manifest's `worlds.mdr.knowledge: ./knowledge` key
 *
 * The file has two passes: a fast CLI-only pass that proves install +
 * check + on-disk layout, and a Docker-gated pass that spins the full
 * 4-agent colony up and verifies the roster.
 */

describe('severance catalog', () => {
    test('init severance installs a check-passing 4-agent project', async () => {
        // Given - an empty dir
        // When - we install the severance gallery entry
        // Then - every expected file lands on disk, knowledge ships
        // At the project root, and the whole project passes check.
        const result = await spec('severance init')
            .project('empty')
            .exec(['init severance', 'check'])
            .run();

        expect(result.exitCode, `stdout:\n${result.stdout.text}`).toBe(0);

        // The four agents all land under spwn/agents/ with their
        // Own SOUL.md. If any agent is missing, the catalog installer
        // Silently regressed.
        for (const name of ['mark', 'helly', 'irving', 'dylan']) {
            expect(
                result.file(`spwn/agents/${name}/agent.yaml`).exists,
                `missing agent.yaml for ${name}`,
            ).toBe(true);
            expect(
                result.file(`spwn/agents/${name}/SOUL.md`).exists,
                `missing SOUL.md for ${name}`,
            ).toBe(true);
        }

        // Knowledge ships at the project root (NOT under spwn/) and
        // The spwn.yaml's `worlds.mdr.knowledge` key points at it.
        expect(result.file('knowledge').exists).toBe(true);
        expect(result.file('spwn.yaml').content).toMatch(/knowledge: \.\/knowledge/);

        // And `spwn check` is clean on the freshly installed tree.
        expect(result.stdout.text).toContain('Project is valid');
    });

    test('severance agents carry distinct SOUL.md content', async () => {
        // Soul content is what differentiates the four Severance
        // Agents in this showcase. If the installer ever shipped one
        // SOUL.md across all four, the demo collapses into a single
        // Voice — this test pins the divergence byte-level.
        const result = await spec('severance souls distinct')
            .project('empty')
            .exec('init severance')
            .run();

        expect(result.exitCode).toBe(0);
        const souls = ['mark', 'helly', 'irving', 'dylan'].map(
            (name) => result.file(`spwn/agents/${name}/SOUL.md`).content,
        );
        // Every pair should be byte-different — a trivial N^2 check
        // For four items.
        for (let i = 0; i < souls.length; i += 1) {
            for (let j = i + 1; j < souls.length; j += 1) {
                expect(souls[i], `souls ${i} and ${j} are byte-identical`).not.toBe(souls[j]);
            }
        }
    });

    test('mdr world spawns all four agents in one container', async () => {
        // Docker-gated smoke: the full 4-agent colony comes up, the
        // World's container is running, and the roster inlined into
        // Each agent's CLAUDE.md names every MDR member. Proves the
        // Severance entry is end-to-end live, not just well-scaffolded.
        await using result = await spec('severance up mdr')
            .project('empty')
            .exec(['init severance', 'up mdr'])
            .run();

        expect(result.exitCode, `stderr:\n${result.stderr.text}`).toBe(0);

        // Colony container lookup is by world config key — for
        // Severance the world is `mdr`, not any individual agent
        // Name. The container hosts all four agents inside.
        const mdr = result.container('mdr');
        expect(mdr.running).toBe(true);

        // Roster is regenerated on every spawn and inlined into each
        // Agent's CLAUDE.md. Pick one agent and assert every member
        // Appears in the prompt.
        const claude = mdr.file('/agents/mark/CLAUDE.md').content;
        for (const name of ['mark', 'helly', 'irving', 'dylan']) {
            expect(claude, `roster missing ${name}`).toContain(name);
        }
    });
});
