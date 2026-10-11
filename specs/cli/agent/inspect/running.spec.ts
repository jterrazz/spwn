import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

// hazard: a document cannot scope the containers a session starts: each session opens on `up`

describe('running', () => {
    test('agent inspect describes the attached agent and its mind', async () => {
        // Given - the project brought up, then its agent inspected
        await using result = await cli
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .exec(['up', 'agent inspect neo']);

        // Then - the description, whole
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch(file('./__expected__/agent-inspect.txt'));
    });
});
