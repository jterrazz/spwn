import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot read a container back

const real = cli.with({ env: { SPWN_BASE_IMAGE: null } });

describe('real-build', () => {
    test('the scaffold, brought up, builds a real image whose tools all probe and whose world runs', async () => {
        // Given - the scaffold brought up on a real image build
        await using result = await real
            .fixture(directory('./__fixtures__/scaffolded-project/'))
            .exec('up');

        // Then - the build and every tool probe pass, and the world runs
        expect({
            exitCode: result.exitCode,
            running: result.containerIds.length > 0,
        }).toStrictEqual({ exitCode: 0, running: true });
    });
});
