import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

// hazard: a document cannot scope the containers a session starts, and two listings are JSON judged by their structure

describe('running', () => {
    test('an agent brought up is listed running in its world', async () => {
        // Given - the project brought up, then its agents listed
        await using result = await cli
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .exec(['up', 'agent ls --json']);

        // Then - neo runs in world neo
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch(file('./__expected__/agents-running.json'));
    });
});
