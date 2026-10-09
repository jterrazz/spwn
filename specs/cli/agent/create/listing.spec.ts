import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

// hazard: a document cannot judge JSON by its structure: the listing is compared whole as the value it parses to

describe('listing', () => {
    test('two created agents are listed unattached', async () => {
        // Given - two agents created in the run's own spwn home, then listed as JSON
        await using result = await cli.exec([
            'agent create neo',
            'agent create trinity',
            'agent ls --json',
        ]);

        // Then - the envelope lists both agents, attached to no world
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            agents: [
                { name: 'neo', status: 'unattached' },
                { name: 'trinity', status: 'unattached' },
            ],
            mode: 'global',
        });
    });
});
