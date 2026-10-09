import { waitUntil } from '@jterrazz/test';
import { execFileSync } from 'node:child_process';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

const MARKER = 'spwn-web-shutdown-home';

const survivorsCarrying = (marker: string): string =>
    execFileSync('pgrep', ['-fl', marker], { encoding: 'utf8' }).trim();

describe('server', () => {
    test('web serves the API until stopped, then leaves no process behind', async () => {
        // Given - the server started in a home whose path marks every process it spawns, then the scope ends and the runner stops it
        {
            await using result = await cli
                .env({ SPWN_HOME: `$WORKDIR/${MARKER}` })
                .exec('web --no-open --port 0', {
                    timeout: 15_000,
                    waitFor: 'spwn API listening on',
                });

            // Then - it reached its listening banner, cut there, and printed no error
            expect(result.exitCode).toBe(0);
            expect(result.stdout).toMatch(`

                  ⬡ spwn web

                  ⚠  Frontend  not available (run: cd apps/web && npm install)

                  ✓  API Server  full
                     http://localhost:0

                  Press Ctrl+C to stop

                spwn API listening on :0
            `);
        }

        // Then - once the OS has reaped, no process carries the marker
        await waitUntil(
            () => {
                try {
                    survivorsCarrying(MARKER);
                    return false;
                } catch {
                    return true;
                }
            },
            { interval: 200, timeout: 5000, why: 'the server left no process carrying the marker' },
        );
    });
});
