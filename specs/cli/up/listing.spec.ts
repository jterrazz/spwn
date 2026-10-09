import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

describe('listing', () => {
    test('a world brought up is listed running with its agent', async () => {
        // Given - the project brought up, then its worlds listed
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'world list --json']);

        // Then - one running world holding neo
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            mode: 'project',
            worlds: [{ agents: ['neo'], name: 'neo', status: 'running' }],
        });
    });

    test('an agent brought up is listed running in its world', async () => {
        // Given - the project brought up, then its agents listed
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'agent ls --json']);

        // Then - neo runs in world neo
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch(file('./__expected__/agents-running.json'));
    });

    test('status shows the running world and its agent', async () => {
        // Given - the project brought up, then the environment's status asked
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'status']);

        // Then - the status report, whole
        expect(result.exitCode).toBe(0);
        // checker-disable-next-line ground-shared-or-inline -- a stream that opens on a line break and shares an indentation cannot be written in place: the comparison dedents such a literal, its quoted form included
        expect(result.stderr).toMatch(file('./__expected__/status.txt'));
    });
});
