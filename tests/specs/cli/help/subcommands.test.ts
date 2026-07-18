import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Subcommand help-text contracts. The top-level help pages are byte-goldened in
 * output.test.ts; this file pins the key subcommand help pages byte-for-byte too.
 * cobra help lives in version-pinned Go strings, so it is stable output like any
 * other — the repo already goldens `check --help` (check.test.ts) and `up --help`
 * (lifecycle/backend-flag.test.ts), so those pages are not re-probed here. Every
 * result binds with `await using`.
 */

describe('subcommand help text', () => {
    test('agent create --help documents the Mind layout', async () => {
        // Given - the agent create help page
        await using result = await cli.fixture('$FIXTURES/empty/').exec('agent create --help');

        // Then - the full usage page matches byte-for-byte
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('agent-create-help.txt');
    });

    test('install --help documents scoping + scheme grammar', async () => {
        // Given - the install help page
        await using result = await cli.fixture('$FIXTURES/empty/').exec('install --help');

        // Then - the full usage page matches byte-for-byte
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('install-help.txt');
    });

    test('init --help advertises the gallery shorthand', async () => {
        // Given - the init help page
        await using result = await cli.fixture('$FIXTURES/empty/').exec('init --help');

        // Then - the full usage page matches byte-for-byte
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('init-help.txt');
    });

    test('skill new --help documents the local authoring flow', async () => {
        // Given - the skill new help page
        await using result = await cli.fixture('$FIXTURES/empty/').exec('skill new --help');

        // Then - the full usage page matches byte-for-byte
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('skill-new-help.txt');
    });

    test('down --help renders the lifecycle usage block', async () => {
        // Given - the down lifecycle help page
        await using result = await cli.fixture('$FIXTURES/empty/').exec('down --help');

        // Then - the full usage page matches byte-for-byte
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch('down-help.txt');
    });
});
