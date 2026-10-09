import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

describe('json', () => {
    test('reports declared agents with their world for a project', async () => {
        // Given - single-agent has one declared agent, neo, in world neo
        await using result = await cli
            .fixture(directory('../../__fixtures__/single-agent/'))
            .exec('agent ls --json');

        // Then - the JSON envelope lists neo as stopped, attached to neo
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            agents: [{ name: 'neo', status: 'stopped', world: 'neo' }],
            mode: 'project',
        });
    });

    test('marks an undeclared agent dir as orphan', async () => {
        // Given - single-agent base + a ghost agent dir that is not in spwn.yaml
        await using result = await cli
            .fixture(directory('../../__fixtures__/single-agent/'))
            .fixture(directory('./__fixtures__/ghost/'))
            .exec('agent ls --json');

        // Then - ghost appears as orphan with no world, neo stays declared
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            agents: [
                { name: 'ghost', status: 'orphan' },
                { name: 'neo', status: 'stopped', world: 'neo' },
            ],
            mode: 'project',
        });
    });

    test('on an empty home returns no agents', async () => {
        // Given - an isolated home with no agents created
        await using result = await cli
            .env({ SPWN_HOME: '$WORKDIR/spwn-home' })
            .exec('agent ls --json');

        // Then - the JSON envelope reports an empty roster
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({ agents: [], mode: 'global' });
    });
});
