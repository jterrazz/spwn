import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

// hazard: a document cannot scope the containers a session starts nor read them back

const simulated = cli.with({ env: { SPWN_BASE_IMAGE: 'spwn-test:latest' } });

describe('ephemeral', () => {
    test('an ephemeral task registers no agent', async () => {
        // Given - the project brought up, a task dispatched, then the agents listed
        await using result = await simulated
            .fixture(directory('../../__fixtures__/docker-pilot/'))
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
