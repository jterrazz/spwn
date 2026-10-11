import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot read a container back nor scope the containers a session starts

describe('colony', () => {
    test('a world of two agents holds both homes, and each prompt names both members', async () => {
        // Given - the docker-pilot project grown to a colony of neo and morpheus, brought up
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .fixture(directory('../__fixtures__/colony/'))
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
});
