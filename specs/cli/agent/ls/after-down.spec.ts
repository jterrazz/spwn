import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

// hazard: a document cannot scope the containers a session starts nor read them back: each session opens on `up`

describe('after-down', () => {
    test('a destroyed world is listed stopped, its agent unattached', async () => {
        // Given - the project brought up, torn down, then its agents listed
        await using result = await cli
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .exec(['up', 'down', 'agent ls --json']);

        // Then - neo is declared, and stopped
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            agents: [{ name: 'neo', status: 'stopped', world: 'neo' }],
            mode: 'project',
        });
    });
});
