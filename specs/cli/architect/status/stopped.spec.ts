import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { ARCHITECT_IMAGE_OVERRIDE, withoutArchitect } from '../../../__setup__/architect.js';
import { cli } from '../../cli.specification.js';

// hazard: a document cannot hold the architect lock the specs share on the one container the product names

const architect = cli.with({ env: { SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE } });

describe('stopped', () => {
    test('a stopped architect reports not running', async () => {
        // Given - the architect started, stopped, then asked for its status
        await using result = await withoutArchitect(
            async () =>
                await architect
                    .fixture(directory('../../__fixtures__/docker-pilot/'))
                    .exec(['architect start', 'architect stop', 'architect status']),
        );

        // Then - the status report finds no container
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch(file('./__expected__/not-running.txt'));
    });
});
