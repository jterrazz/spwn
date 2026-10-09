import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

describe('json', () => {
    test('a project lists its declared worlds, stopped', async () => {
        // Given - single-agent declares one world neo with one agent
        await using result = await cli
            .fixture(directory('../../__fixtures__/single-agent/'))
            .exec('world list --json');

        // Then - the JSON envelope lists the declared world in stopped state
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            mode: 'project',
            worlds: [{ agents: ['neo'], name: 'neo', status: 'stopped' }],
        });
    });
});
