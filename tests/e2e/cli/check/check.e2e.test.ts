import { fileURLToPath } from 'node:url';
import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';
import { stdoutMatcher } from '../../../setup/fixtures.js';

/**
 * Stage 2a of the @jterrazz/test migration: per-feature folder layout
 * with real stdout fixtures.
 *
 * Snapshots live under ./expected/stdout/<name>.txt. Regenerate them
 * with `JTERRAZZ_TEST_UPDATE=1 pnpm -C tests exec vitest run e2e/cli/check`.
 *
 * Temp-dir paths are normalised to `<PROJECT>` inside the matcher so
 * fixtures stay stable across runs and machines (see setup/fixtures.ts).
 */

const TEST_FILE = fileURLToPath(import.meta.url);

describe('spwn check', () => {
    test('valid project prints a clean success report', async () => {
        // Given - the frozen single-agent fixture (one agent, one world)
        const result = await spec('check valid').project('single-agent').exec('check').run();

        // Then - exits zero with the canonical "Project is valid" banner
        expect(result.exitCode).toBe(0);
        await stdoutMatcher(TEST_FILE, result.stdout).toMatchFixture('valid-project');
    });

    test('--help prints the check command usage', async () => {
        // Given - any cwd; --help is resolved before the project walk
        const result = await spec('check help').project('empty').exec('check --help').run();

        // Then - cobra emits the usage block for the check subcommand
        expect(result.exitCode).toBe(0);
        await stdoutMatcher(TEST_FILE, result.stdout).toMatchFixture('help');
    });

    test('flags an agent that references a non-existent built-in tool', async () => {
        // Given - check-invalid-tool has tools: ["@spwn/nonexistent"]
        const result = await spec('check invalid tool')
            .project('check-invalid-tool')
            .exec('check')
            .run();

        // Then - exits non-zero and lists the built-ins the user can pick from
        expect(result.exitCode).not.toBe(0);
        await stdoutMatcher(TEST_FILE, result.stdout).toMatchFixture('invalid-tool-ref');
    });

    test('flags a remote-registry tool reference as unsupported', async () => {
        // Given - check-registry-tool has tools: ["@jterrazz/foo"]
        const result = await spec('check registry tool')
            .project('check-registry-tool')
            .exec('check')
            .run();

        // Then - exits non-zero with the "remote registries not yet supported" rule
        expect(result.exitCode).not.toBe(0);
        await stdoutMatcher(TEST_FILE, result.stdout).toMatchFixture('registry-not-supported');
    });

    test('seed overlay flags the one-agent-one-world rule', async () => {
        // Given - single-agent base + a seed fragment that adds a second
        // World claiming the same neo agent. The framework's seed handler
        // Merges the YAML fragment into spwn.yaml inside the temp project.
        const result = await spec('one-agent-one-world')
            .project('single-agent')
            .seed('spwn.yaml/two-worlds-same-agent.yaml')
            .exec('check')
            .run();

        // Then - check fails with the rule violation, and the second
        // World is genuinely present in spwn.yaml on disk (proves the
        // Seed handler ran).
        expect(result.exitCode).not.toBe(0);
        expect(result.file('spwn.yaml').content).toContain('duplicate:');
        const combined = result.stdout.text + result.stderr.text;
        // The rule fires with "agent X already deployed by world Y" — the
        // Exact wording lives in the validator. We assert on intent, not
        // Wording, by also checking the manifest path the violation
        // Points at.
        expect(combined).toContain('already deployed by world "duplicate"');
        expect(combined).toContain('spwn.yaml#worlds.neo.agents');
    });

    test('errors when run outside a spwn project', async () => {
        // Given - the empty fixture has no spwn.yaml anywhere up the tree
        const result = await spec('check no project').project('empty').exec('check').run();

        // Then - exits non-zero and nudges the user at spwn init
        expect(result.exitCode).not.toBe(0);
        const combined = (result.stdout.text + result.stderr.text).toLowerCase();
        expect(combined).toContain('spwn init');
        expect(combined).toContain('spwn.yaml');
    });
});
