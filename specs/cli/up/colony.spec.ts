import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot read a container back nor scope the containers a session starts

describe('colony', () => {
    test('a world of two agents holds both homes, and each prompt names both members', async () => {
        // Given - the docker-pilot project grown to a colony of neo and morpheus, brought up
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .fixture(directory('./__fixtures__/colony/'))
            .exec('up');

        // Then - one running container with both homes, and neo's roster lists both
        expect(result.exitCode).toBe(0);
        const neo = result.container('neo');
        const inside = await neo.exec(
            'sh -c \'ls /agents && sed -n "/^### Members/,/^$/p" /agents/neo/CLAUDE.md\'',
        );
        expect({ inside: inside.stdout.text, running: neo.running }).toStrictEqual({
            inside: 'morpheus\nneo\n### Members\n- **morpheus** (chief)\n- **neo** (worker)\n\n',
            running: true,
        });
    });

    test('the colony is listed as one world holding both agents', async () => {
        // Given - the colony brought up, then its worlds listed
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .fixture(directory('./__fixtures__/colony/'))
            .exec(['up', 'world list --json']);

        // Then - one running world, both agents in it
        expect(result.exitCode).toBe(0);
        expect(result.json).toMatch({
            mode: 'project',
            worlds: [
                {
                    name: 'neo',
                    status: 'running',
                    agents: ['morpheus', 'neo'],
                },
            ],
        });
    });

    test('tearing the colony down removes the one container and keeps both minds', async () => {
        // Given - the colony brought up, then torn down
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .fixture(directory('./__fixtures__/colony/'))
            .exec(['up', 'down']);

        // Then - no container, both souls on disk
        expect(result.exitCode).toBe(0);
        expect({
            containers: result.containerIds,
            morpheus: result.file('spwn/agents/morpheus/SOUL.md').exists,
            neo: result.file('spwn/agents/neo/SOUL.md').exists,
        }).toStrictEqual({ containers: [], morpheus: true, neo: true });
    });
});
