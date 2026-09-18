import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn agent ls` — every spec whose subject is the JSON envelope rather
 * than the printed roster. These stay in code because the comparison is
 * STRUCTURAL: `expect(result.json).toMatch(…)` judges the envelope by
 * shape against `_expected/*.json`, which a byte-exact `stdout:` cannot
 * express. The header spec compares two SEPARATE runs to each other,
 * which a document — one binary, one working directory, one session —
 * has no vocabulary for either.
 *
 * The `ghost/` overlay seeds an undeclared agent dir to exercise the
 * orphan path. The runner is docker-aware, so every result binds with
 * `await using` (rule B5).
 */

// Column spacing varies with row content, so collapse runs of whitespace to a single space before comparing the header row.
const extractAgentLsHeader = (txt: string) =>
    txt
        .split('\n')
        .find((l) => /\bAGENT\b/.test(l) && /\bSTATUS\b/.test(l))
        ?.trim()
        .replaceAll(/\s+/g, ' ');

describe('agent ls --json', () => {
    test('reports declared agents with their world for a project', async () => {
        // Given - single-agent has one declared agent, neo, in world neo
        await using result = await cli.fixture('$FIXTURES/single-agent/').exec('agent ls --json');

        // Then - the JSON envelope lists neo as stopped, attached to neo
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch('declared.json');
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

    test('lists created agents structurally', async () => {
        // Given - two agents created then listed as JSON in an isolated home
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec(['agent create neo', 'agent create trinity', 'agent ls --json']);

        // Then - the JSON envelope lists both agents unattached
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch('ls-two-agents.json');
    });

    test('on an empty home returns no agents', async () => {
        // Given - an isolated home with no agents created
        await using result = await cli
            .fixture('$FIXTURES/empty/')
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec('agent ls --json');

        // Then - the JSON envelope reports an empty roster
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch('ls-empty.json');
    });

    test('the header is stable across project and global mode', async () => {
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
        expect(globalHeader).toStrictEqual(projectHeader);
    });
});
