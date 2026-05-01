import { describe, expect, test } from 'vitest';

import { spec } from '../../../_setup/cli.specification.js';

// Column spacing varies with row content, so collapse runs of whitespace to a single space before comparing the header row.
const extractAgentLsHeader = (txt: string) =>
    txt
        .split('\n')
        .find((l) => /\bAGENT\b/.test(l) && /\bSTATUS\b/.test(l))
        ?.trim()
        .replace(/\s+/g, ' ');

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

    test('agent ls header is stable across project and global mode', async () => {
        // Given - global mode (no project active) and project mode both render a table; the column schema must match so users don't see the header jump when they cd into a project.
        const global = await spec('agent ls global')
            .project('empty')
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec(['agent new neo', 'agent ls'])
            .run();

        const project = await spec('agent ls project')
            .project('single-agent')
            .exec('agent ls')
            .run();

        expect(global.exitCode).toBe(0);
        expect(project.exitCode).toBe(0);

        // Then - the header row (the line containing AGENT) has the same column ordering in both outputs.
        const globalHeader = extractAgentLsHeader(global.stderr.text);
        const projectHeader = extractAgentLsHeader(project.stderr.text);
        expect(globalHeader).toBeDefined();
        expect(projectHeader).toBeDefined();
        expect(globalHeader).toEqual(projectHeader);
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
