import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot read a container back nor scope the containers a session starts

describe('colony', () => {
    test('tearing the colony down removes the one container and keeps both minds', async () => {
        // Given - the colony brought up, then torn down
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .fixture(directory('../__fixtures__/colony/'))
            .exec(['up', 'down']);

        // Then - no container, both souls on disk
        expect(result.exitCode).toBe(0);
        expect({
            containers: result.containerIds,
            morpheus: result.file('spwn/agents/morpheus/SOUL.md').exists,
            neo: result.file('spwn/agents/neo/SOUL.md').exists,
        }).toStrictEqual({ containers: [], morpheus: true, neo: true });
    });
});
