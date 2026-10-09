import { directory, file } from '@jterrazz/test';
import { spawnSync } from 'node:child_process';
import { afterEach, describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

const WORLD = '$(ls .spwn/world-states | head -1)';

describe('snapshots', () => {
    afterEach(() => {
        const tags = spawnSync(
            'docker',
            [
                'images',
                '--format',
                '{{.Repository}}:{{.Tag}}',
                '--filter',
                'reference=spwn-snapshot:*--round-trip',
            ],
            { encoding: 'utf8' },
        )
            .stdout.split('\n')
            .filter((tag) => tag !== '');
        spawnSync('docker', ['rmi', '-f', ...tags], { encoding: 'utf8' });
    });

    test('a saved snapshot is listed under its name', async () => {
        // Given - the project brought up, a snapshot saved, then the snapshots listed
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec(['up', `world snap save ${WORLD} --name round-trip`, 'world snap ls']);

        // Then - the listing, whole
        expect(result.exitCode).toBe(0);
        // checker-disable-next-line ground-shared-or-inline -- a stream that opens on a line break and shares an indentation cannot be written in place: the comparison dedents such a literal, its quoted form included
        expect(result.stderr).toMatch(file('./__expected__/snapshot-listed.txt'));
    });

    test('a removed snapshot leaves the listing', async () => {
        // Given - the project brought up, a snapshot saved then removed, then the snapshots listed
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .exec([
                'up',
                `world snap save ${WORLD} --name round-trip`,
                `world snap rm ${WORLD}--round-trip`,
                'world snap ls',
            ]);

        // Then - the listing, whole
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toMatch('  ✓ No snapshots.\n');
    });
});
