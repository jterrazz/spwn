import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn world list --json` coverage.
 *
 * The text path is exercised via execution/status snapshots; this file
 * focuses on the JSON envelope so the declared-world merge (name,
 * status, agents) stays structurally pinned independent of human
 * formatting.
 */

describe('spwn world list --json', () => {
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
