import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { ARCHITECT_IMAGE_OVERRIDE, withoutArchitect } from '../../../__setup__/architect.js';
import { cli } from '../../cli.specification.js';

// hazard: a document cannot hold the architect lock the specs share on the one container the product names

describe('idle', () => {
    test('stop with no architect running is a clean no-op', async () => {
        // Given - a stop issued with no architect container up
        await using result = await withoutArchitect(
            async () =>
                await cli
                    .fixture(directory('../../__fixtures__/docker-pilot/'))
                    .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
                    .exec('architect stop'),
        );

        // Then - it reports nothing to stop
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch(file('./__expected__/nothing-to-stop.txt'));
    });
});
