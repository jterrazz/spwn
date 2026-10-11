import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot scope the containers a session starts, and two listings are JSON judged by their structure

describe('running', () => {
    test('status shows the running world and its agent', async () => {
        // Given - the project brought up, then the environment's status asked
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'status']);

        // Then - the status report, whole
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch(file('./__expected__/status.txt'));
    });
});
