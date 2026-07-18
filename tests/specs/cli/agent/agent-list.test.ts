import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn agent ls --json` coverage.
 *
 * The text path covers the empty-home case in the crud suite. Here we
 * pin the project-mode JSON envelope, including the declared/orphan
 * distinction and the per-agent status mapping. The `ghost/` overlay
 * seeds an undeclared agent dir to exercise the orphan path. The runner
 * is docker-aware, so every result binds with `await using` (rule B5).
 */

// Column spacing varies with row content, so collapse runs of whitespace to a single space before comparing the header row.
const extractAgentLsHeader = (txt: string) =>
    txt
        .split('\n')
        .find((l) => /\bAGENT\b/.test(l) && /\bSTATUS\b/.test(l))
        ?.trim()
        .replace(/\s+/g, ' ');

describe('agent ls --json', () => {
    test('reports declared agents with their world for a project', async () => {
        // Given - single-agent has one declared agent, neo, in world neo
        await using result = await cli.fixture('$FIXTURES/single-agent/').exec('agent ls --json');

        // Then - the JSON envelope lists neo as stopped, attached to neo
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch('declared.json');
    });

    test('agent ls header is stable across project and global mode', async () => {
        // Given - global mode (isolated home) and project mode both render a table
        await using global = await cli
            .fixture('$FIXTURES/empty/')
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec(['agent new neo', 'agent ls']);
        await using project = await cli.fixture('$FIXTURES/single-agent/').exec('agent ls');

        expect(global.exitCode).toBe(0);
        expect(project.exitCode).toBe(0);

        // Then - the header row (the line containing AGENT) has the same column ordering in both outputs
        const globalHeader = extractAgentLsHeader(global.stderr.text);
        const projectHeader = extractAgentLsHeader(project.stderr.text);
        expect(globalHeader).toBeDefined();
        expect(projectHeader).toBeDefined();
        expect(globalHeader).toEqual(projectHeader);
    });

    test('marks an undeclared agent dir as orphan', async () => {
        // Given - single-agent base + a ghost agent dir that is not in spwn.yaml
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .fixture('ghost/')
            .exec('agent ls --json');

        // Then - ghost appears as orphan with no world, neo stays declared
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch('with-orphan.json');
    });
});
