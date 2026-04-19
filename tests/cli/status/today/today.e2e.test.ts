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
    test('agent create writes the Mind structure under spwn/agents/', async () => {
        // Given - the single-agent fixture already has `neo`; we create
        // A second agent alongside it to exercise the command without
        // Clobbering the frozen fixture state.
        const result = await spec('agent create mind')
            .project('single-agent')
            .exec('agent create trinity')
            .run();

        // Then - exits zero and creates SOUL.md + the two Mind layer
        // Directories on disk. identity/ was collapsed into SOUL.md at
        // The agent root in 2026-04; knowledge is world-scoped, not a
        // Mind layer; skills moved to build-time deps at /world/skills/.
        expect(result.exitCode).toBe(0);
        expect(result.file('spwn/agents/trinity/SOUL.md').exists).toBe(true);
        for (const layer of ['playbooks', 'journal']) {
            expect(result.file(`spwn/agents/trinity/${layer}`).exists, `missing ${layer}/`).toBe(
                true,
            );
        }
        expect(result.file('spwn/agents/trinity/skills').exists).toBe(false);
        expect(result.file('spwn/agents/trinity/identity').exists).toBe(false);
        expect(result.file('spwn/agents/trinity/knowledge').exists).toBe(false);

        // The starter persona is seeded into SOUL.md
        const persona = result.file('spwn/agents/trinity/SOUL.md');
        expect(persona.exists).toBe(true);
        expect(persona.content.length).toBeGreaterThan(10);

        // Legacy layout names must not be recreated.
        expect(result.file('spwn/agents/trinity/core').exists).toBe(false);
        expect(result.file('spwn/agents/trinity/memory').exists).toBe(false);
        expect(result.file('spwn/agents/trinity/sessions').exists).toBe(false);
    });
});

describe('CLI upgrade', () => {
    test('upgrade --check queries the release feed and prints a version', async () => {
        // Given - the upgrade command hits the GitHub API to learn
        // About the latest release. The latest version tag changes
        // Over time and the network may be flaky on CI, so we do a
        // Substring match on the stable portions of the banner
        // Rather than a full snapshot.
        const result = await spec('upgrade check').project('empty').exec('upgrade --check').run();

        expect(result.exitCode).toBe(0);
        const stderr = result.stderr.text;
        expect(stderr).toContain('Current version');
        expect(stderr).toContain('Checking for updates');
    });
});
