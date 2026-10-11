import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

// hazard: a document cannot scope the containers a session starts nor read them back: each session opens on `up`

describe('after-down', () => {
    test('a destroyed world is listed stopped', async () => {
        // Given - the project brought up, torn down, then its worlds listed
        await using result = await cli
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .exec(['up', 'down', 'world list --json']);

        // Then - the declared world is back to stopped
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            mode: 'project',
            worlds: [{ agents: ['neo'], name: 'neo', status: 'stopped' }],
        });
    });
});
