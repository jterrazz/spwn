import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

// hazard: a document cannot scope the containers a session starts, and two listings are JSON judged by their structure

describe('running', () => {
    test('a world brought up is listed running with its agent', async () => {
        // Given - the project brought up, then its worlds listed
        await using result = await cli
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .exec(['up', 'world list --json']);

        // Then - one running world holding neo
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            mode: 'project',
            worlds: [{ agents: ['neo'], name: 'neo', status: 'running' }],
        });
    });
});
