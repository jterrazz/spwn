import { describe, expect, test } from 'vitest';

import { spec } from '../../../_setup/cli.specification.js';

/**
 * `spwn world list --json` coverage.
 *
 * The text path is exercised via execution/status snapshots; this file
 * focuses on the JSON envelope so the declared-world merge (name,
 * status, agents) stays structurally pinned independent of human
 * formatting.
 */

describe('spwn world list --json', () => {
    test('world ls is an alias for world list', async () => {
        // Given - single-agent declares one world `neo`
        // When - running the short form
        // Then - same JSON envelope as `world list --json`
        const viaLs = await spec('world ls alias')
            .project('single-agent')
            .exec('world ls --json')
            .run();
        const viaList = await spec('world list canonical')
            .project('single-agent')
            .exec('world list --json')
            .run();

        expect(viaLs.exitCode).toBe(0);
        expect(viaList.exitCode).toBe(0);
        expect(viaLs.stdout.text).toBe(viaList.stdout.text);
    });

    test('reports declared worlds for a project', async () => {
        // Given - single-agent declares one world `neo` with one agent
        const result = await spec('world list json project')
            .project('single-agent')
            .exec('world list --json')
            .run();

        // Then - JSON envelope lists the declared world in stopped state
        expect(result.exitCode).toBe(0);
        await result.json.toMatch('project.json');
    });
});
