import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * Real-build smoke tests: every shipped scaffold must produce a world that
 * `spwn up` can bring online from a cold, empty project. These intentionally
 * do NOT set SPWN_BASE_IMAGE, so the full content-addressed image build runs
 * (generate Dockerfile, install apt packages, probe every declared tool inside
 * the container). They are excluded from the default suite and run via
 * vitest.smoke.config.ts (serialized — they share the spwn-world:latest tag).
 */
function smokeTest(args: string) {
    test(`spwn init ${args || '(default)'} + spwn up produces a running world`, async () => {
        // Given - an empty project scaffolded (optionally from a catalog ref) then brought up with a real image build
        const initArgs = args ? `init ${args}` : 'init';
        await using result = await cli.fixture('$FIXTURES/empty/').exec([initArgs, 'up']);

        // Then - the build + tool-probe path succeeds (a zero exit is itself the assertion) and any captured world is running
        expect(
            result.exitCode,
            `stdout:\n${result.stdout.text}\nstderr:\n${result.stderr.text}`,
        ).toBe(0);
        const neo = result.container('neo');
        if (neo.exists) {
            expect(neo.running).toBe(true);
        }
    });
}

describe('smoke: default scaffold', () => {
    smokeTest('');
});

describe('smoke: catalog examples', () => {
    smokeTest('spwn:matrix');
    smokeTest('spwn:startup');
    smokeTest('spwn:paperclip-factory');
    smokeTest('spwn:research-lab');
    smokeTest('spwn:macrohard');
});
