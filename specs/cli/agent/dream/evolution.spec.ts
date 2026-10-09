import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../../cli.specification.js';

describe('evolution', () => {
    test('dream, sleep, fork and export leave a forked mind and an archive', async () => {
        // Given - neo dreamed, slept, forked into neo-v2 and exported, in one session
        await using result = await cli
            .fixture(directory('../../__fixtures__/docker-pilot/'))
            .exec([
                'agent dream neo',
                'agent sleep neo',
                'agent fork neo neo-v2',
                'agent export neo',
            ]);

        // Then - the session ends cleanly with the fork's mind and the archive on disk
        expect({
            archive: result.file('neo.tar.gz').exists,
            exitCode: result.exitCode,
            forkedSoul: result.file('spwn/agents/neo-v2/SOUL.md').exists,
        }).toStrictEqual({ archive: true, exitCode: 0, forkedSoul: true });
    });
});
