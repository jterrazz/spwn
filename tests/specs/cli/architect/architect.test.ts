import { spawnSync } from 'node:child_process';
import { beforeAll, describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * `spwn architect <start|stop|status>` under docker-aware mode. The
 * Architect is spwn's always-on orchestration container. Unlike world
 * containers it is NOT project-scoped — it lives at a fixed name
 * (`spwn-architect`) with the label `sh.spwn.kind=architect` rather than
 * `sh.spwn.world.config=<name>`, so `result.container(...)` name lookup
 * won't find it; state is read back via follow-up `architect status`
 * invocations instead. spwn still applies the test-run label, so the
 * scoped cleanup at `await using` teardown removes anything spawned here
 * (rule B5). We point spwn at `alpine:latest` via SPWN_ARCHITECT_IMAGE so
 * start never has to build the full architect image.
 *
 * Dropped (with rationale): the legacy `talk --output-format stream-json`
 * smoke test only asserted "does not panic" because claude isn't
 * installed in the alpine stub — no signal; the real streaming path is
 * covered by the talk happy-path in agent/talk.
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
        // Ensure the override image is present locally (skip silently if docker is unavailable)
        if (!dockerImageExists(ARCHITECT_IMAGE_OVERRIDE)) {
            spawnSync('docker', ['pull', ARCHITECT_IMAGE_OVERRIDE], {
                encoding: 'utf8',
                timeout: 60_000,
            });
        }
        // Nuke any leftover architect container from a prior run; the fixed name would collide with start
        spawnSync('docker', ['rm', '-f', 'spwn-architect'], {
            encoding: 'utf8',
            timeout: 10_000,
        });
    });

    test('status reports not running when no architect container exists', async () => {
        // Given - a project with no architect container up
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec('architect status');

        // Then - status reports idle (scalpel: the status line carries a dynamic image/id)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('not running');
    });

    test('start provisions a running architect container', async () => {
        // Given - architect started then re-queried (only the last exec's streams are exposed)
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec(['architect start', 'architect status']);

        // Then - status sees it running against the override image (scalpel: dynamic status line)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('running');
        expect(result.stderr).toContain(ARCHITECT_IMAGE_OVERRIDE);
    });

    test('start is idempotent — second start reports already running', async () => {
        // Given - architect started twice in one chain
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec(['architect start', 'architect start']);

        // Then - the second start is a no-op naming the running container (scalpel: dynamic status line)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('already running');
    });

    test('stop tears the architect container down', async () => {
        // Given - architect started, stopped, then re-queried
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec(['architect start', 'architect stop', 'architect status']);

        // Then - after stop the status reports idle (scalpel: dynamic status line)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('not running');
    });

    test('stop when nothing is running is a clean no-op', async () => {
        // Given - a stop issued with no architect container up
        await using result = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .env({ SPWN_ARCHITECT_IMAGE: ARCHITECT_IMAGE_OVERRIDE })
            .exec('architect stop');

        // Then - it reports idle without panicking (scalpel: dynamic status line + absence probe)
        expect(result.exitCode).toBe(0);
        expect(result.stderr).toContain('not running');
        expect(result.stderr).not.toContain('panic');
    });
});
