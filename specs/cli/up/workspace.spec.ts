import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot run a command inside the running world, which is what proves the mount

describe('workspace', () => {
    test('the project is mounted read-write: the agent reads the host files and what it writes lands on the host', async () => {
        // Given - the docker-pilot project brought up
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec('up');

        // When - the agent reads the manifest and writes a file in its workspace
        const inside = await result
            .container('neo')
            .exec(
                "sh -c 'cd /workspaces/* && head -3 spwn.yaml && echo written > from-the-agent.txt'",
            );

        // Then - it read the host's manifest, and the host holds its file
        expect(result.exitCode).toBe(0);
        expect({
            host: result.file('from-the-agent.txt').content,
            read: inside.stdout.text,
        }).toStrictEqual({
            host: 'written\n',
            read: '# spwn - project manifest\nversion: 1\nname: docker-pilot\n',
        });
    });
});
