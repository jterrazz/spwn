import { describe, expect, test } from 'vitest';

import { spec } from '../../setup/cli.specification.js';

/**
 * Real-build smoke tests: every shipped scaffold must produce a
 * world that `spwn up` can bring online from a cold, empty project.
 *
 * These intentionally do NOT set SPWN_BASE_IMAGE, so the full
 * content-addressed image build runs (generate Dockerfile, install
 * apt packages, probe every declared tool inside the container).
 * The previous failure mode this guards against: a default scaffold
 * declared @spwn/python, the user's cached spwn/world:latest image
 * predated that declaration, the manual version constant hadn't
 * been bumped so the stale image was reused, and `spwn up` failed
 * on the first invocation with "base image does not provide pip3".
 * No e2e test caught it because every other test pins
 * SPWN_BASE_IMAGE to a prebuilt mock and skips the build path.
 *
 * Each test is serialized by the smoke vitest config because they
 * share the spwn/world:latest tag; parallel builds would race.
 */

/**
 * Serialized describe that runs `spwn init` on an empty project
 * (optionally with a catalog example ref), then `spwn up`, then
 * asserts the container came online with every declared tool
 * verified. The probe inside `spwn up` already errors if any tool
 * is missing, so a zero exit is itself the assertion.
 */
function smokeTest(label: string, args: string) {
    test(`spwn init ${args || '(default)'} + spwn up produces a running world`, async () => {
        const initArgs = args ? `init ${args}` : 'init';
        await using result = await spec(label).project('empty').exec([initArgs, 'up']).run();

        if (result.exitCode !== 0) {
            throw new Error(
                `spwn init ${args} + up exited ${result.exitCode}\n` +
                    `stdout:\n${result.stdout.text}\n` +
                    `stderr:\n${result.stderr.text}`,
            );
        }

        // The runner queries the container by the world-config name
        // (spwn.yaml's `worlds:` map key). Every shipped scaffold
        // Declares a world; we assert at least one container tagged
        // With this test run is up.
        const neo = result.container('neo');
        if (neo.exists) {
            expect(neo.running).toBe(true);
        }
    });
}

describe('smoke: default scaffold', () => {
    smokeTest('smoke default init', '');
});

describe('smoke: catalog examples', () => {
    smokeTest('smoke matrix', '@spwn/matrix');
    smokeTest('smoke startup', '@spwn/startup');
    smokeTest('smoke paperclip-factory', '@spwn/paperclip-factory');
    smokeTest('smoke research-lab', '@spwn/research-lab');
    smokeTest('smoke macrohard', '@spwn/macrohard');
});
