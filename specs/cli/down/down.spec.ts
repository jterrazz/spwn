import { directory, file, text } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot scope the containers a session starts nor read them back: each session opens on `up`

describe('down', () => {
    test('down destroys the container and journals the session, the mind kept on disk', async () => {
        // Given - the project brought up, then torn down
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'down']);

        // Then - the teardown session, no container, the soul kept, and the journal holding the session's entry and its record
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch(file('./__expected__/down.txt'));
        expect({
            containers: result.containerIds,
            soul: result.file('spwn/agents/neo/SOUL.md').exists,
        }).toStrictEqual({ containers: [], soul: true });
        const journal = await result.directory('spwn/agents/neo/journal').files();
        expect(text(journal.map((name) => `${name}\n`).join(''))).toMatch(`
            .gitkeep
            {{date}}_{{number}}_world-neo-{{hex#world}}.md
            world-neo-{{hex#world}}.json
        `);
    });

    test('a second down finds nothing to destroy and says so', async () => {
        // Given - the project brought up, then torn down twice
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'down', 'down']);

        // Then - the second teardown, whole
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch(file('./__expected__/second-down.txt'));
    });

    test('a down naming a destroyed world is refused without a stack or the usage page', async () => {
        // Given - the project brought up, torn down, then the world named again
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'down', 'down neo']);

        // Then - the refusal alone
        expect(result.exitCode).toBe(1);
        expect(result.stderr).toMatch(file('./__expected__/down-destroyed.txt'));
    });

    test('a down by world id prints each step of the teardown', async () => {
        // Given - the project brought up, then torn down by the id spwn recorded for it
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'down $(ls .spwn/world-states | head -1)']);

        // Then - the per-step teardown, whole
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch(file('./__expected__/down-by-id.txt'));
    });
});
