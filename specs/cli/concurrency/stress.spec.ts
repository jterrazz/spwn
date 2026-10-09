import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

describe('stress', () => {
    test('spwn up refuses to run when another up is holding the lock', async () => {
        // Given - a stale per-world lockfile already present under .spwn/ (a concurrent up that has not released)
        await using result = await cli
            .fixture(directory('../__fixtures__/docker-pilot/'))
            .fixture(directory('./__fixtures__/lock-held/'))
            .exec('up neo');

        // Then - exits non-zero with the in-progress message and no container was created
        expect(result.exitCode).toBe(1);
        expect(result.stderr.text).toContain('Up in progress');
        expect(result.container('neo').exists).toBe(false);
    });
});
