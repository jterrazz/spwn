import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

// hazard: a document cannot read a container back: its labels, mounts, network and agent home are the claim

describe('second-up', () => {
    test('a second up of a running world adds no second world', async () => {
        // Given - the project brought up twice, then listed
        await using result = await cli
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .exec(['up', 'up', 'world list --json']);

        // Then - one running world
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            mode: 'project',
            worlds: [{ agents: ['neo'], name: 'neo', status: 'running' }],
        });
    });
});
