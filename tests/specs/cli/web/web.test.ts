import { execSync } from 'node:child_process';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn web` — the long-running Web UI process. `--help` is a one-shot
 * exec; the server itself runs via `.exec(..., { waitFor })`, which runs
 * the child until stdout/stderr matches the readiness banner and then
 * SIGTERM-kills it at scope exit. The `spwn API listening on` banner is
 * emitted once both frontend and API are ready. We bind `--port 0` to
 * dodge collisions and `--no-open` to keep the browser out of CI. The
 * orphan check shells out to `pgrep` — process-tree inspection is genuine
 * plumbing not expressible via the framework. Every result binds with
 * `await using` (rule B5).
 */

const isolated = () => cli.fixture('$FIXTURES/empty/').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('spwn web', () => {
    test('--help documents the core flags', async () => {
        // Given - a one-shot help invocation
        await using result = await isolated().exec('web --help');

        // Then - the usage lists the core flags (scalpel: cobra-formatted help wording, D11(e))
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('--port');
        expect(result.stdout).toContain('--no-open');
    });

    test('web child exits cleanly on SIGTERM, leaves no orphans', async () => {
        // Given - a uniquely-marked SPWN_HOME so surviving processes carrying this env can be grepped after dispose
        const homeMarker = `spwn-test-web-sigterm-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

        {
            // When - the server reaches the listening banner, then the scope exits and the framework SIGTERMs the child
            await using result = await cli
                .fixture('$FIXTURES/empty/')
                .env({ SPWN_HOME: `$WORKDIR/${homeMarker}` })
                .exec('web --no-open --port 0', {
                    timeout: 15_000,
                    waitFor: 'spwn API listening on',
                });

            // Then - the server really started (otherwise the orphan check below is trivially satisfied)
            expect(result.exitCode).toBe(0);
            expect(result.stdout).toContain('spwn API listening on');
        }

        // Give the OS a brief moment to reap; retry a few times before failing outright
        let orphans = 'not checked';
        for (let i = 0; i < 5; i++) {
            try {
                orphans = execSync(`pgrep -fl "${homeMarker}" || true`, {
                    encoding: 'utf8',
                }).trim();
            } catch {
                orphans = '';
            }
            // Filter our own pgrep match and bare shell wrappers whose argv carries the marker via their env
            orphans = orphans
                .split('\n')
                .filter((line) => line && !line.includes('pgrep'))
                .filter((line) => !/^\s*\d+\s+sh\s*$/.test(line))
                .filter((line) => !/^\s*\d+\s+(?:sh|bash)\s+-c\s/.test(line))
                .join('\n');
            if (orphans === '') {
                break;
            }
            await new Promise((resolve) => setTimeout(resolve, 200));
        }
        expect(orphans).toBe('');
    });

    test('starts the API server and reaches the listening banner', async () => {
        // Given - the server started with the readiness banner as the wait target
        await using result = await isolated().exec('web --no-open --port 0', {
            timeout: 15_000,
            waitFor: 'spwn API listening on',
        });

        // Then - waitFor fires (exit 0; 124 would mean the banner never appeared) with no runtime errors on stderr
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('spwn API listening on');
        expect(result.stderr.text).not.toContain('TypeError');
        expect(result.stderr.text).not.toContain('ReferenceError');
        expect(result.stderr.text).not.toContain('panic:');
    });
});
