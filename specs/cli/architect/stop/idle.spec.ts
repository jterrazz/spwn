import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';
import { ARCHITECT_IMAGE_OVERRIDE, withoutArchitect } from '../__setup__/architect.js';

describe('idle', () => {
    test('stop with no architect running is a clean no-op', async () => {
        // Given - a stop issued with no architect container up
        withoutArchitect();
        await using result = await cli
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec('architect stop');

        // Then - it reports nothing to stop
        expect(result.exitCode).toBe(0);
        // checker-disable-next-line ground-shared-or-inline -- a stream that opens on a line break and shares an indentation cannot be written in place: the comparison dedents such a literal, its quoted form included
        expect(result.stderr).toMatch(file('./__expected__/nothing-to-stop.txt'));
    });
});
