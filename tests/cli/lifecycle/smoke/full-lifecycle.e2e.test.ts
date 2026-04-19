import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Non-Docker agent lifecycle: create -> inspect -> dream -> rm.
 *
 * The legacy test wove the Docker world-spawn flow (up / world inspect
 * / down) into the same journey; those steps are covered by the
 * Docker-gated lifecycle tests under `tests/cli/lifecycle/` and are
 * intentionally omitted here. This file exercises everything a user
 * can do without spinning up a container.
 */

describe('agent lifecycle (CLI-only)', () => {
    test('init -> agent ls -> dream -> rm round-trip', async () => {
        // Given - an empty dir scaffolded with the starter project
        // When - we run the full non-Docker journey in one chain
        const result = await spec('lifecycle round trip')
            .project('empty')
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec(['init --name lifecycle', 'agent ls', 'agent dream neo', 'agent rm neo'])
            .run();

        // Then - final `agent rm neo` exits zero with the delete banner
        expect(result.exitCode).toBe(0);
        expect(result.stderr.text).toMatch(/Deleted agent\s+neo/);

        // And the agent directory is gone from the scaffolded project
        expect(result.file('spwn/agents/neo').exists).toBe(false);
    });

    test('init scaffolds the default agent with the expected mind layout', async () => {
        const result = await spec('lifecycle mind layout')
            .project('empty')
            .exec('init --name layout-check')
            .run();
        expect(result.exitCode).toBe(0);

        // Project-mode scaffold writes SOUL.md + the two Mind layer
        // Directories (playbooks/journal) under spwn/agents/neo.
        // Identity/ was collapsed into SOUL.md in 2026-04; skills moved
        // To build-time dependencies at /world/skills/. Knowledge is
        // World-scoped, opt-in via the worlds.<name>.knowledge key —
        // Default scaffold creates the flat ./knowledge/ directory at
        // The project root.
        expect(result.file('spwn/agents/neo').exists).toBe(true);
        expect(result.file('spwn/agents/neo/identity').exists).toBe(false);
        expect(result.file('spwn/agents/neo/skills').exists).toBe(false);
        expect(result.file('spwn/agents/neo/playbooks').exists).toBe(true);
        expect(result.file('spwn/agents/neo/journal').exists).toBe(true);
        expect(result.file('spwn/agents/neo/knowledge').exists).toBe(false);
        expect(result.file('spwn/agents/neo/SOUL.md').exists).toBe(true);
        expect(result.file('spwn/agents/neo/agent.yaml').exists).toBe(true);
        expect(result.file('knowledge').exists).toBe(true);
        expect(result.file('spwn/worlds').exists).toBe(false);
    });

    test('multiple agents can coexist in the same project', async () => {
        const result = await spec('lifecycle multi agent')
            .project('empty')
            .exec(['init --name multi-agent', 'agent create trinity', 'agent ls'])
            .run();
        expect(result.exitCode).toBe(0);

        // Both the scaffolded `neo` and the freshly created `trinity`
        // Land on disk
        expect(result.file('spwn/agents/neo').exists).toBe(true);
        expect(result.file('spwn/agents/trinity').exists).toBe(true);

        // And `agent ls` lists both names (rendered to stderr)
        result.stderr.toContain('neo');
        result.stderr.toContain('trinity');
    });
});
