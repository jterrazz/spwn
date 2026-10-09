import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

const simulated = cli.with({ env: { SPWN_BASE_IMAGE: 'spwn-test:latest' } });

describe('ephemeral', () => {
    test('an ephemeral task dispatched into a running world runs and leaves the world up', async () => {
        // Given - the project brought up, then a task dispatched into its world by id
        await using result = await simulated
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec([
                'up',
                'agent --ephemeral "lint the code" --world $(ls .spwn/world-states | head -1)',
            ]);

        // Then - the dispatch, whole, and the world still running
        expect(result.exitCode).toBe(0);
        // checker-disable-next-line ground-shared-or-inline -- a stream that opens on a line break and shares an indentation cannot be written in place: the comparison dedents such a literal, its quoted form included
        expect(result.stderr).toMatch(file('./__expected__/ephemeral.txt'));
        expect(result.container('neo').running).toBe(true);
    });

    test('an ephemeral task registers no agent', async () => {
        // Given - the project brought up, a task dispatched, then the agents listed
        await using result = await simulated
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec([
                'up',
                'agent --ephemeral "check health" --world $(ls .spwn/world-states | head -1)',
                'agent ls --json',
            ]);

        // Then - neo alone
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch(file('./__expected__/agents-running.json'));
    });
});
