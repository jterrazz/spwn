import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';
import { ARCHITECT_IMAGE_OVERRIDE, withoutArchitect } from '../__setup__/architect.js';

describe('idle', () => {
    test('status reports not running when no architect container exists', async () => {
        // Given - a project with no architect container up
        withoutArchitect();
        await using result = await cli
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec('architect status');

        // Then - the status report finds no container
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch(file('../__expected__/not-running.txt'));
    });
});
