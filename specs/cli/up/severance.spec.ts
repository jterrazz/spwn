import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

// hazard: a document cannot read a container back: the agent homes are listed inside the running world

describe('severance', () => {
    test('the mdr world of the severance example runs its four agents in one container', async () => {
        // Given - the severance example installed, then its mdr world brought up
        await using result = await cli.exec(['init severance', 'up mdr']);

        // Then - one running container holds the four agent homes
        expect(result.exitCode).toBe(0);
        const mdr = result.container('mdr');
        const homes = await mdr.exec('ls /agents');
        expect({ homes: homes.stdout.text, running: mdr.running }).toStrictEqual({
            homes: 'dylan\nhelly\nirving\nmark\n',
            running: true,
        });
    });
});
