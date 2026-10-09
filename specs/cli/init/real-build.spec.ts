import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot run a table of examples nor read a container back

const real = cli.with({ env: { SPWN_BASE_IMAGE: null } });

describe('real-build', () => {
    test.each([
        '',
        'spwn:matrix',
        'spwn:startup',
        'spwn:paperclip-factory',
        'spwn:research-lab',
        'spwn:macrohard',
    ])(
        'init %s, then up, builds a real image whose tools all probe and whose world runs',
        async (example) => {
            // Given - a scaffold, or a catalog example, brought up on a real image build
            await using result = await real.exec([`init ${example}`.trim(), 'up']);

            // Then - the build and every tool probe pass, and the world runs
            expect({
                exitCode: result.exitCode,
                running: result.containerIds.length > 0,
            }).toStrictEqual({ exitCode: 0, running: true });
        },
    );
});
