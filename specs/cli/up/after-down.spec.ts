import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot scope the containers a session starts nor read them back: each session opens on `up`

describe('after-down', () => {
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
