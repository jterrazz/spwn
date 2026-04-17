import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * Coverage for the lockfile-consistency rule in `spwn check`.
 *
 * The rule only fires once a lockfile exists — projects without one
 * (legacy or brand-new) silently pass. After running `spwn tool
 * install` the lockfile shows up and any drift in agent.yaml against
 * it surfaces as an error.
 */
describe('spwn check — lockfile consistency', () => {
    test('silent when no lockfile exists', async () => {
        // The docker-pilot fixture ships without a lockfile; the rule is a no-op.
        const result = await spec('check no lockfile').project('docker-pilot').exec('check').run();

        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('Project is valid');
    });
});

describe('spwn check — lockfile drift', () => {
    test('passes after installing every declared ref', async () => {
        const result = await spec('check matches lockfile')
            .project('docker-pilot')
            .exec(['install spwn:unix', 'install spwn:git', 'check'])
            .run();

        expect(result.exitCode).toBe(0);
        expect(result.stdout.text).toContain('Project is valid');
    });

    test('flags drift when lockfile is incomplete', async () => {
        const result = await spec('check detects drift')
            .project('docker-pilot')
            .exec(['install spwn:unix', 'check'])
            .run();

        expect(result.exitCode).not.toBe(0);
        // Canonical scheme form in user-facing output.
        expect(result.stdout.text + result.stderr.text).toContain('spwn:git');
        expect(result.stdout.text + result.stderr.text).toMatch(/lock/i);
    });
});
