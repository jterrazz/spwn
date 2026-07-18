import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn inspect` prints a kubectl-describe / cargo-tree blend: a per-agent
 * block with an identity header, a resolved dependency tree, a skills list,
 * and a hooks list. Tests use --offline so the output is fully deterministic
 * (no live world-status lookup), which lets the full render be a golden. Every
 * result binds with `await using` (rule B5); no docker.
 */

describe('spwn inspect', () => {
    test('renders a single-agent project end-to-end', async () => {
        // Given - the frozen single-agent fixture (one agent, three spwn: deps)
        await using result = await cli.fixture('$FIXTURES/single-agent/').exec('inspect --offline');

        // Then - exits clean and the block matches the golden
        expect(result.exitCode, result.stderr.text).toBe(0);
        expect(result.stdout).toMatch('single-agent.txt');
    });

    test('focuses on a single named agent', async () => {
        // Given - inspect scoped to the neo agent
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .exec('inspect neo --offline');

        // Then - exactly one agent block renders (scalpel: header count is a structural probe)
        expect(result.exitCode).toBe(0);
        const headerCount = (result.stdout.text.match(/(?:^|\n)Name\s+/g) ?? []).length;
        expect(headerCount).toBe(1);
        expect(result.stdout).toContain('Name         neo');
    });

    test('errors cleanly when the named agent is missing', async () => {
        // Given - inspect targeting an agent that does not exist
        await using result = await cli
            .fixture('$FIXTURES/single-agent/')
            .exec('inspect ghost --offline');

        // Then - exits non-zero naming the missing agent (error-message probe)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toContain('agent "ghost" not found');
    });

    test('errors when run outside a spwn project', async () => {
        // Given - the empty fixture has no spwn.yaml up the tree
        await using result = await cli.fixture('$FIXTURES/empty/').exec('inspect --offline');

        // Then - exits non-zero with the no-project error (error-message probe)
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toContain('no spwn.yaml found');
    });

    test('renders a local tool dependency without the local: prefix', async () => {
        // Given - mixed-tool-refs declares tool/my-local-tool on disk under spwn/tools/
        await using result = await cli
            .fixture('$FIXTURES/mixed-tool-refs/')
            .exec('inspect --offline');

        // Then - the local tool shows as a bare name, not "local:my-local-tool" (absence probe)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('my-local-tool');
        expect(result.stdout).not.toContain('local:my-local-tool');
    });

    test('--help prints the inspect command usage', async () => {
        // Given - the inspect help page
        await using result = await cli.fixture('$FIXTURES/empty/').exec('inspect --help');

        // Then - the usage names the command and the --offline flag (cobra help probe)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('inspect');
        expect(result.stdout).toContain('--offline');
    });
});
