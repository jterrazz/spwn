import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../../cli.specification.js';

// hazard: a document cannot scope the containers a session starts: the session opens on `up`

describe('knowledge', () => {
    test('world knowledge ls lists what the world knowledge folder holds', async () => {
        // Given - a project whose knowledge folder holds a note, brought up, then the world's knowledge listed by id
        await using result = await cli
            .fixture(directory('../../../__fixtures__/docker-pilot/'))
            .fixture(directory('./__fixtures__/noted-knowledge/'))
            .exec(['up', 'world knowledge ls $(ls .spwn/world-states | head -1)']);

        // Then - the listing, whole
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toMatch(
            '  Knowledge files:\n    note.md\n  Use "spwn world knowledge show <world-id> <path>" to read a file.\n',
        );
    });
});
