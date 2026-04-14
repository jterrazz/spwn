import { spawnSync } from 'node:child_process';
import { beforeAll, describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * `spwn architect <start|stop|status|talk>` under the docker() spec mode.
 *
 * The Architect is spwn's always-on orchestration container. Unlike world
 * containers it is NOT project-scoped — it lives at a fixed name
 * (`spwn-architect`) and carries the label `sh.spwn.kind=architect` rather
 * than `sh.spwn.world.config=<name>`. As a result the framework's
 * `result.container('spwn-architect')` name lookup won't find it (the
 * runner resolves names against `sh.spwn.world.config`). We rely on:
 *   - follow-up `spwn architect status` invocations for state checks
 *   - the test-run label (`sh.spwn.test.run`) which spwn applies to the
 *     architect container via labels.ApplyTestRun, so the scoped cleanup
 *     at `await using` teardown still removes anything this test spawned
 *
 * To keep the suite fast and hermetic we always point spwn at
 * `alpine:latest` via `SPWN_ARCHITECT_IMAGE`. Otherwise start would try
 * to build the full architect image (cross-compiled spwn + claude tools)
 * from the source tree, which is both slow and infeasible outside a dev
 * checkout. The alpine image just needs to stay alive (sleep infinity)
 * so status/stop have something to inspect; we don't exercise the
 * inside-the-container claude runtime here.
 *
 * Dropped (with rationale):
 *   - `talk --output-format stream-json outputs JSON events`: the legacy
 *     test only asserted "does not panic" because claude isn't installed
 *     in the alpine stub. That's a no-signal smoke test; the actual JSON
 *     streaming path is covered by the talk happy-path in agent/talk.
 */

const ARCHITECT_IMAGE_OVERRIDE = 'alpine:latest';

function dockerImageExists(image: string): boolean {
    const res = spawnSync('docker', ['image', 'inspect', image], {
        encoding: 'utf8',
        timeout: 5000,
    });
    return res.status === 0;
}

describe('spwn architect', () => {
    beforeAll(() => {
        // Ensure the override image is present locally. Skip silently if
        // Docker is unavailable — vitest will fail the individual tests
        // With a clearer error from the spec runner.
        if (!dockerImageExists(ARCHITECT_IMAGE_OVERRIDE)) {
            spawnSync('docker', ['pull', ARCHITECT_IMAGE_OVERRIDE], {
                encoding: 'utf8',
                timeout: 60_000,
            });
        }
        // Nuke any leftover architect container from a prior run (dev
        // Machine state, interrupted runs). The fixed name `spwn-architect`
        // Would otherwise collide with this suite's start commands.
        spawnSync('docker', ['rm', '-f', 'spwn-architect'], {
            encoding: 'utf8',
            timeout: 10_000,
        });
    });

    test('status reports not running when no architect container exists', async () => {
        await using result = await spec('architect status idle')
            .project('docker-pilot')
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec('architect status')
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('not running');
    });

    test('start provisions a running architect container', async () => {
        await using result = await spec('architect start')
            .project('docker-pilot')
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec(['architect start', 'architect status'])
            .run();

        expect(result.exitCode).toBe(0);
        // Only the LAST command's streams are exposed on multi-exec, so
        // The visible output here is from `architect status`. That means
        // Both "start" succeeded (exit 0) and status sees it running.
        result.stderr.toContain('running');
        // When overriding the image we see alpine in the status line,
        // Not the default spwn/architect:latest.
        result.stderr.toContain(ARCHITECT_IMAGE_OVERRIDE);
    });

    test('start is idempotent — second start reports already running', async () => {
        await using result = await spec('architect start twice')
            .project('docker-pilot')
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec(['architect start', 'architect start'])
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('already running');
    });

    test('stop tears the architect container down', async () => {
        await using result = await spec('architect stop')
            .project('docker-pilot')
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec(['architect start', 'architect stop', 'architect status'])
            .run();

        expect(result.exitCode).toBe(0);
        // Last command is `status` — after stop it must report idle.
        result.stderr.toContain('not running');
    });

    test('stop when nothing is running is a clean no-op', async () => {
        await using result = await spec('architect stop idle')
            .project('docker-pilot')
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec('architect stop')
            .run();

        expect(result.exitCode).toBe(0);
        result.stderr.toContain('not running');
        expect(result.stderr.text).not.toContain('panic');
    });
});
