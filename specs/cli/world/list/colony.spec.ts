import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

// hazard: a document cannot read a container back nor scope the containers a session starts

describe('colony', () => {
    test('the colony is listed as one world holding both agents', async () => {
        // Given - the colony brought up, then its worlds listed
        await using result = await cli
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .fixture(directory('../../__fixtures__/colony/'))
            .exec(['up', 'world list --json']);

        // Then - one running world, both agents in it
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            mode: 'project',
            worlds: [
                {
                    name: 'neo',
                    status: 'running',
                    agents: ['morpheus', 'neo'],
                },
            ],
        });
    });
});
