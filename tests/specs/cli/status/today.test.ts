import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Project-mode agent-scaffolding and upgrade-banner checks. Agents live under
 * `spwn/agents/<name>/` inside the project; the upgrade banner is probed for
 * its stable lines only (the release tag and network state are dynamic). Every
 * result binds with `await using` (rule B5); these are CLI-only.
 */

describe('agent mind structure', () => {
    test('agent create writes the Mind structure under spwn/agents/', async () => {
        // Given - the single-agent fixture; create a second agent alongside frozen neo
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .exec('agent create trinity');

        // Then - exits zero and writes SOUL.md plus the two Mind-layer dirs
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

        // And - the starter persona is seeded into SOUL.md
        const persona = result.file('spwn/agents/trinity/SOUL.md');
        expect(persona.exists).toBe(true);
        expect(persona.content.length).toBeGreaterThan(10);

        // And - legacy layout names must not be recreated
        expect(result.file('spwn/agents/trinity/core').exists).toBe(false);
        expect(result.file('spwn/agents/trinity/memory').exists).toBe(false);
        expect(result.file('spwn/agents/trinity/sessions').exists).toBe(false);
    });
});

describe('cli upgrade', () => {
    test('upgrade --check queries the release feed and prints a version', async () => {
        // Given - the upgrade command hits the GitHub release feed
        await using result = await cli.fixture('$FIXTURES/empty/').exec('upgrade --check');

        // Then - the stable banner lines render (scalpel: latest tag + network state are dynamic)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('Current version');
        expect(result.stderr).toContain('Checking for updates');
    });
});
