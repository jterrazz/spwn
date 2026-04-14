import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * System-level CLI tests that exercise today's project-mode behaviour.
 *
 * The legacy suite here covered global-mode `spwn init` + `~/.spwn/`
 * scaffolding, which has been removed. What remains is rewritten to
 * the modern project shape: agents live under `spwn/agents/<name>/`
 * inside the project, not under a user-home tree.
 */

describe('agent mind structure', () => {
    test('agent create writes the 5-layer Mind structure under spwn/agents/', async () => {
        // Given - the single-agent fixture already has `neo`; we create
        // A second agent alongside it to exercise the command without
        // Clobbering the frozen fixture state.
        const result = await spec('agent create mind')
            .project('single-agent')
            .exec('agent create Trinity')
            .run();

        // Then - exits zero and creates the 5 Mind layers on disk.
        expect(result.exitCode).toBe(0);
        for (const layer of ['core', 'skills', 'knowledge', 'playbooks', 'journal']) {
            expect(result.file(`spwn/agents/Trinity/${layer}`).exists, `missing ${layer}/`).toBe(
                true,
            );
        }

        // The starter persona is seeded into core/profile.md
        const persona = result.file('spwn/agents/Trinity/core/profile.md');
        expect(persona.exists).toBe(true);
        expect(persona.content.length).toBeGreaterThan(10);

        // The legacy (pre-Mind) layout must not be recreated.
        expect(result.file('spwn/agents/Trinity/identity').exists).toBe(false);
        expect(result.file('spwn/agents/Trinity/memory').exists).toBe(false);
        expect(result.file('spwn/agents/Trinity/sessions').exists).toBe(false);
    });
});

describe('CLI upgrade', () => {
    test('upgrade --check exits cleanly', async () => {
        // Given - the upgrade command hits the GitHub API to learn
        // About the latest release. The output banner is written to
        // Stderr and discarded on success by the ExecAdapter, so we
        // Can only assert on the exit code here — the point of the
        // Test is that `upgrade --check` itself does not crash.
        const result = await spec('upgrade check').project('empty').exec('upgrade --check').run();

        expect(result.exitCode).toBe(0);
    });
});
