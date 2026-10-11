import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot scope the containers a session starts nor read them back

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
        expect(result.stderr).toMatch(file('./__expected__/ephemeral.txt'));
        expect(result.container('neo').running).toBe(true);
    });
});
