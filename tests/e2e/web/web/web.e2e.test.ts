import { describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

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
        const combined = result.stdout.text + result.stderr.text;
        expect(combined).toContain('spwn API listening on');
        expect(combined).not.toContain('TypeError');
        expect(combined).not.toContain('ReferenceError');
        expect(combined).not.toContain('panic:');
    });
});
