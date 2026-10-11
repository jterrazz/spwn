import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { ARCHITECT_IMAGE_OVERRIDE, withoutArchitect } from '../../../__setup__/architect.js';
import { cli } from '../../cli.specification.js';

// hazard: a document cannot hold the architect lock nor read the container back: the product names its one container whatever the home

const architect = cli.with({ env: { SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE } });

describe('lifecycle', () => {
    test('start provisions the architect container on the override image', async () => {
        // Given - the architect started
        await using result = await withoutArchitect(
            async () =>
                await architect
                    .fixture(directory('../../__fixtures__/docker-pilot/'))
                    .exec('architect start'),
        );

        // Then - the session names the container it created
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch(file('./__expected__/started.txt'));
    });

    test('a second start reports the architect already running', async () => {
        // Given - the architect started twice
        await using result = await withoutArchitect(
            async () =>
                await architect
                    .fixture(directory('../../__fixtures__/docker-pilot/'))
                    .exec(['architect start', 'architect start']),
        );

        // Then - the second start changes nothing and says so
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch(file('./__expected__/already-running.txt'));
    });
});
