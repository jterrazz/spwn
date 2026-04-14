import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn agent ls --json` coverage.
 *
 * The text path covers the empty-home case in cli/execution. Here we
 * pin the project-mode JSON envelope, including the declared/orphan
 * distinction and the per-agent status mapping.
 */

describe('spwn agent ls --json', () => {
    test('reports declared agents with their world for a project', async () => {
        // Given - single-agent has one declared agent, neo, in world neo
        const result = await spec('agent ls json declared')
            .project('single-agent')
            .exec('agent ls --json')
            .run();

        // Then - the JSON envelope lists neo as stopped, attached to neo
        expect(result.exitCode).toBe(0);
        await result.json.toMatch('declared.json');
    });

    test('marks an undeclared agent dir as orphan', async () => {
        // Given - single-agent + a seeded ghost agent that is not in spwn.yaml
        const result = await spec('agent ls json orphan')
            .project('single-agent')
            .seed('agent/ghost')
            .exec('agent ls --json')
            .run();

        // Then - ghost appears as orphan with no world, neo stays declared
        expect(result.exitCode).toBe(0);
        await result.json.toMatch('with-orphan.json');
    });
});
