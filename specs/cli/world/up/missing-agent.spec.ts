import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

// hazard: a document cannot list the containers a session leaves, and the refusal must leave none

describe('missing-agent', () => {
    test('world up naming an agent that does not exist is refused, with the command that creates it', async () => {
        // Given - a world up naming an agent that does not exist
        await using result = await cli
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .exec('world up --agent ghost');

        // Then - non-zero with the "agent new" hint (full stderr golden) and no container left behind
        expect(result.exitCode).toBe(1);
        // checker-disable-next-line ground-shared-or-inline -- a stream that opens on a line break and shares an indentation cannot be written in place: the comparison dedents such a literal, its quoted form included
        expect(result.stderr).toMatch(file('./__expected__/missing-agent.txt'));
        expect(result.containerIds).toStrictEqual([]);
    });
});
