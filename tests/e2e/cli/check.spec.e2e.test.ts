import { describe, expect, test } from 'vitest';

import { spec } from '../../setup/cli.specification.js';

/**
 * Stage 1 proof of concept for the @jterrazz/test migration.
 *
 * Same coverage shape as the legacy ctx.spwn(...) helpers, but written
 * against the new fluent runner. Demonstrates:
 *  - .project('name') copies a fixture into a fresh temp dir
 *  - .exec('args') runs the binary in that dir
 *  - result.exitCode + result.stdout (no custom output helpers needed)
 *  - result.file(path) for asserting on disk artifacts
 *
 * Compare the diff vs tests/e2e/cli/refs.e2e.test.ts (the closest
 * legacy equivalent) to evaluate the shape before rolling the rest of
 * the suite.
 */
describe('spwn check (new spec runner)', () => {
    test('passes on a freshly initialised project', async () => {
        // Given - the single-agent fixture: spwn init's exact output, frozen on disk
        const result = await spec('check single-agent').project('single-agent').exec('check').run();

        // Then - check exits zero and reports the project as valid
        expect(result.exitCode).toBe(0);
        const out = result.stdout + result.stderr;
        expect(out).toContain('Project is valid');
    });

    test('lists check rules in --help', async () => {
        // Given - any fixture (--help is fixture-independent but the runner needs a cwd)
        const result = await spec('check help').project('empty').exec('check --help').run();

        // Then - cobra prints the check command's usage
        expect(result.exitCode).toBe(0);
        expect(result.stdout.toLowerCase()).toContain('check');
    });

    test('errors when run outside a project', async () => {
        // Given - the empty fixture has no spwn.yaml
        const result = await spec('check no project').project('empty').exec('check').run();

        // Then - check fails and points the user at spwn init
        expect(result.exitCode).not.toBe(0);
        const out = (result.stdout + result.stderr).toLowerCase();
        expect(out).toMatch(/spwn\.yaml|spwn init/);
    });

    test('the single-agent fixture really has spwn.yaml on disk', async () => {
        // Given - a no-op exec just to materialise the project copy
        const result = await spec('verify single-agent layout')
            .project('single-agent')
            .exec('check')
            .run();

        // Then - spwn.yaml + the neo agent dir are present in the temp cwd
        expect(result.file('spwn.yaml').exists).toBe(true);
        expect(result.file('spwn.yaml').content).toContain('worlds:');
        expect(result.file('spwn/agents/neo/core/profile.md').exists).toBe(true);
    });
});
