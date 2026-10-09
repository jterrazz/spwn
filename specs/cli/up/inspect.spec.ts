import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot scope the containers a session starts: each session opens on `up`

describe('inspect', () => {
    test('world inspect describes the running world', async () => {
        // Given - the project brought up, then its world inspected by id
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'world inspect $(ls .spwn/world-states | head -1)']);

        // Then - the description, whole
        expect(result.exitCode).toBe(0);
        // checker-disable-next-line ground-shared-or-inline -- a stream that opens on a line break and shares an indentation cannot be written in place: the comparison dedents such a literal, its quoted form included
        expect(result.stderr).toMatch(file('./__expected__/world-inspect.txt'));
    });

    test('agent inspect describes the attached agent and its mind', async () => {
        // Given - the project brought up, then its agent inspected
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'agent inspect neo']);

        // Then - the description, whole
        expect(result.exitCode).toBe(0);
        // checker-disable-next-line ground-shared-or-inline -- a stream that opens on a line break and shares an indentation cannot be written in place: the comparison dedents such a literal, its quoted form included
        expect(result.stderr).toMatch(file('./__expected__/agent-inspect.txt'));
    });
});
