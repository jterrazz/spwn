import { directory, file } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

describe('down', () => {
    test('down destroys the container and journals the session, the mind kept on disk', async () => {
        // Given - the project brought up, then torn down
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'down']);

        // Then - the teardown session, no container, and one journal entry beside the soul
        expect(result.exitCode).toBe(0);
        // checker-disable-next-line ground-shared-or-inline -- a stream that opens on a line break and shares an indentation cannot be written in place: the comparison dedents such a literal, its quoted form included
        expect(result.stderr).toMatch(file('./__expected__/down.txt'));
        const journal = await result.directory('spwn/agents/neo/journal').files();
        expect({
            containers: result.containerIds,
            journal: journal.length,
            soul: result.file('spwn/agents/neo/SOUL.md').exists,
        }).toStrictEqual({ containers: [], journal: 3, soul: true });
    });

    test('a destroyed world is listed stopped, its agent unattached', async () => {
        // Given - the project brought up, torn down, then its agents listed
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'down', 'agent ls --json']);

        // Then - neo is declared, and stopped
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            agents: [{ name: 'neo', status: 'stopped', world: 'neo' }],
            mode: 'project',
        });
    });

    test('a destroyed world is listed stopped', async () => {
        // Given - the project brought up, torn down, then its worlds listed
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'down', 'world list --json']);

        // Then - the declared world is back to stopped
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            mode: 'project',
            worlds: [{ agents: ['neo'], name: 'neo', status: 'stopped' }],
        });
    });

    test('a second down finds nothing to destroy and says so', async () => {
        // Given - the project brought up, then torn down twice
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'down', 'down']);

        // Then - the second teardown, whole
        expect(result.exitCode).toBe(0);
        // checker-disable-next-line ground-shared-or-inline -- a stream that opens on a line break and shares an indentation cannot be written in place: the comparison dedents such a literal, its quoted form included
        expect(result.stderr).toMatch(file('./__expected__/second-down.txt'));
    });

    test('a down naming a destroyed world is refused without a stack or the usage page', async () => {
        // Given - the project brought up, torn down, then the world named again
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'down', 'down neo']);

        // Then - the refusal alone
        expect(result.exitCode).toBe(1);
        // checker-disable-next-line ground-shared-or-inline -- a stream that opens on a line break and shares an indentation cannot be written in place: the comparison dedents such a literal, its quoted form included
        expect(result.stderr).toMatch(file('./__expected__/down-destroyed.txt'));
    });

    test('a down by world id prints each step of the teardown', async () => {
        // Given - the project brought up, then torn down by the id spwn recorded for it
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'down $(ls .spwn/world-states | head -1)']);

        // Then - the per-step teardown, whole
        expect(result.exitCode).toBe(0);
        // checker-disable-next-line ground-shared-or-inline -- a stream that opens on a line break and shares an indentation cannot be written in place: the comparison dedents such a literal, its quoted form included
        expect(result.stderr).toMatch(file('./__expected__/down-by-id.txt'));
    });

    test('a world can be brought up again after a down, and leaves no codex state behind', async () => {
        // Given - the project brought up, torn down, then up again
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', 'down', 'up']);

        // Then - one running container and no stray .codex in the agent's home
        expect(result.exitCode).toBe(0);
        expect({
            codex: result.file('spwn/agents/neo/.codex').exists,
            running: result.container('neo').running,
        }).toStrictEqual({ codex: false, running: true });
    });
});
