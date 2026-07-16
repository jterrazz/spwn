import { execSync } from 'node:child_process';
import { describe, expect, test } from 'vitest';

import { spec } from '../../../_setup/cli.specification.js';

/**
 * `spwn web` — long-running Web UI process.
 *
 * `--help` is a one-shot exec. The actual server is started via the
 * framework's `.spawn(...)` mode, which runs the child until stdout
 * or stderr contains `waitFor` and then SIGTERM-kills it. The banner
 * `spwn API listening on` is emitted after both the frontend and the
 * API are ready, so it is a solid readiness marker.
 *
 * We bind the API to `--port 0` so we never collide with a real
 * server, and `--no-open` keeps the browser out of CI.
 */

const isolated = (label: string) =>
    spec(label).project('empty').env({ SPWN_HOME: '$WORKDIR/spwn-home' });

describe('spwn web', () => {
    test('--help documents the core flags', async () => {
        const result = await isolated('web help').exec('web --help').run();

        expect(result.exitCode).toBe(0);
        const out = result.stdout.text;
        expect(out).toContain('--port');
        expect(out).toContain('--no-open');
    });

    test('web child exits cleanly on SIGTERM, leaves no orphans', async () => {
        // Given - a uniquely-marked SPWN_HOME so we can grep for any
        // Surviving processes carrying this test's env after dispose.
        // When - the spec scope exits, the framework SIGTERMs the spawned
        // Child.
        // Then - no processes referencing the marker should remain,
        // Proving `spwn web` honours SIGTERM and tears down its children.
        const homeMarker = `spwn-test-web-sigterm-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

        {
            await using result = await spec('web sigterm')
                .project('empty')
                .env({ SPWN_HOME: `$WORKDIR/${homeMarker}` })
                .spawn('web --no-open --port 0', {
                    timeout: 15_000,
                    waitFor: 'spwn API listening on',
                })
                .run();

            expect(result.exitCode).toBe(0);
            // Prove we actually reached the listening banner (i.e. the
            // Server really started) — otherwise the orphan check below
            // Is trivially satisfied because nothing ever ran.
            result.stdout.toContain('spwn API listening on');
        }

        // Give the OS a brief moment to reap; retry a few times before
        // Failing the assertion outright.
        let orphans = 'not checked';
        for (let i = 0; i < 5; i++) {
            try {
                orphans = execSync(`pgrep -fl "${homeMarker}" || true`, {
                    encoding: 'utf8',
                }).trim();
            } catch {
                orphans = '';
            }
            // Filter out our own ripgrep/shell match and bare shell
            // Wrappers whose argv happens to contain the marker via
            // Their env var (pgrep -f matches the full command line
            // Plus env on some systems, so the parent `sh -c ...`
            // That launched the spec child gets caught even after
            // Its child exited).
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
        const result = await isolated('web listening')
            .spawn('web --no-open --port 0', {
                timeout: 15_000,
                waitFor: 'spwn API listening on',
            })
            .run();

        // Pattern-matched spawn resolves with exitCode 0 once waitFor
        // Fires; 124 means the timeout expired before the banner
        // Appeared.
        expect(result.exitCode).toBe(0);
        result.stdout.toContain('spwn API listening on');
        expect(result.stderr.text).not.toContain('TypeError');
        expect(result.stderr.text).not.toContain('ReferenceError');
        expect(result.stderr.text).not.toContain('panic:');
    });
});
