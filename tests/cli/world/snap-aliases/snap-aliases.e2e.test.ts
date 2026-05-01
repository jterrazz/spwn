import { spawnSync } from 'node:child_process';
import { afterEach, describe, expect, test } from 'vitest';

import { spec } from '../../../_setup/cli.specification.js';

/**
 * World snapshot CLI under the docker() spec mode.
 *
 * Renamed from "snap-aliases": the legacy test exclusively exercised the
 * top-level `spwn snap <subcmd>` alias. That alias was removed — snap
 * now lives at `spwn world snap <subcmd>` only (see apps/cli/root.go
 * attaching snap.Cmd to world.Cmd). Asserting on an alias that no longer
 * exists would test nothing, so these tests target the real command
 * path instead.
 *
 * Two architectural notes that shape how these tests are written:
 *
 *   1. `spwn world snap save` takes a full world *id* (not a world key
 *      like "neo"). In project mode the id is generated at spawn time;
 *      we read it off the container's label map via inspect and pass
 *      it into a follow-up spec call. The first run's container
 *      stays alive until its `await using` scope exits, so the second
 *      run finds it via docker labels.
 *
 *   2. Snapshots are docker images (tagged `spwn-snapshot:<id>--<name>`)
 *      that persist on the daemon across test runs. An afterEach hook
 *      force-removes every `spwn-snapshot:*` image so state does not
 *      leak between tests.
 *
 * Dropped:
 *   - "snap restore" — restore spawns a fresh world from the image,
 *     producing a new world id. In project mode that new world carries
 *     our SPWN_TEST_LABEL so cleanup works, but the flow is long and
 *     the value over the lifecycle test is marginal. Re-add if the
 *     image-backed restore path regresses.
 */

function cleanupSnapshotImages(): void {
    try {
        const list = spawnSync(
            'docker',
            [
                'images',
                '--format',
                '{{.Repository}}:{{.Tag}}',
                '--filter',
                'reference=spwn-snapshot:*',
            ],
            { encoding: 'utf8' },
        );
        const tags = (list.stdout || '').trim().split('\n').filter(Boolean);
        for (const tag of tags) {
            spawnSync('docker', ['rmi', '-f', tag], { encoding: 'utf8' });
        }
    } catch {
        // Best-effort cleanup — don't fail the suite if docker rmi fails.
    }
}

describe('world snap', () => {
    afterEach(() => {
        cleanupSnapshotImages();
    });

    test('`spwn world snap --help` lists the save/ls/restore/rm subcommands', async () => {
        await using result = await spec('world snap help')
            .project('docker-pilot')
            .exec('world snap --help')
            .run();

        expect(result.exitCode).toBe(0);
        // Cobra --help renders on stdout.
        result.stdout.toContain('save');
        result.stdout.toContain('ls');
        result.stdout.toContain('restore');
        result.stdout.toContain('rm');
    });

    test('`spwn world snap save` + `snap ls` + `snap rm` round-trip', async () => {
        // Bring up a neo world and learn its runtime world id from the
        // Container labels. The first run's container stays alive while
        // This scope is open, so follow-up spwn commands in a second
        // DockerSpec call can reach it via docker labels.
        await using up = await spec('snap cycle up').project('docker-pilot').exec('up').run();

        expect(up.exitCode).toBe(0);

        const neo = up.container('neo');
        expect(neo.running).toBe(true);

        const inspectData = neo.inspect.value as {
            Config?: { Labels?: Record<string, string> };
        };
        const worldId = inspectData.Config?.Labels?.['sh.spwn.world.id'];
        expect(worldId).toBeTruthy();
        expect(typeof worldId).toBe('string');

        // Save a snapshot by id.
        await using save = await spec('snap cycle save')
            .project('docker-pilot')
            .exec(`world snap save ${worldId} --name round-trip`)
            .run();

        expect(save.exitCode).toBe(0);
        // Save banner + snapshot name land on stderr.
        expect(save.stderr.text).toMatch(/Saved snapshot|Snapshot/i);
        save.stderr.toContain('round-trip');

        // List snapshots — our tag must appear.
        await using list = await spec('snap cycle ls')
            .project('docker-pilot')
            .exec('world snap ls')
            .run();

        expect(list.exitCode).toBe(0);
        // `snap ls` renders a table on stderr (ui.Table default writer).
        list.stderr.toContain('round-trip');

        // Remove the snapshot.
        await using rm = await spec('snap cycle rm')
            .project('docker-pilot')
            .exec(`world snap rm ${worldId}--round-trip`)
            .run();

        expect(rm.exitCode).toBe(0);

        // Confirm it's gone.
        await using after = await spec('snap cycle ls after rm')
            .project('docker-pilot')
            .exec('world snap ls')
            .run();

        expect(after.exitCode).toBe(0);
        // After rm the snapshot must be absent from both streams.
        expect(after.stderr.text).not.toContain('round-trip');
        expect(after.stdout.text).not.toContain('round-trip');
    });
});
