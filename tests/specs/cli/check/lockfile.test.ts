import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Lockfile-consistency rule in `spwn check`. The rule only fires once a lockfile
 * exists; drift between agent.yaml and the lockfile then surfaces as an error.
 * The clean-report golden lives in check.test.ts.
 */

test('silent when no lockfile exists', async () => {
    // Given - docker-pilot ships without a lockfile, so the rule is a no-op
    await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('check');

    // Then - the project is reported valid
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('Project is valid');
});

test('passes after installing every declared ref', async () => {
    // Given - both declared deps installed before checking
    await using result = await cli
        .fixture('$FIXTURES/docker-pilot/')
        .exec(['install spwn:unix', 'install spwn:git', 'check']);

    // Then - the lockfile matches the manifest and check passes
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain('Project is valid');
});

test('flags drift when lockfile is incomplete', async () => {
    // Given - only one of the two declared deps installed
    await using result = await cli
        .fixture('$FIXTURES/docker-pilot/')
        .exec(['install spwn:unix', 'check']);

    // Then - check fails naming the missing dep (scalpel: drift wording spans stdout/stderr)
    expect(result.exitCode).not.toBe(0);
    const combined = result.stdout.text + result.stderr.text;
    expect(combined).toContain('spwn:git');
    expect(combined).toMatch(/lock/i);
});
