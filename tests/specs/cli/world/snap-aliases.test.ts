import { spawnSync } from 'node:child_process';
import { afterEach, describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * World snapshot CLI under docker-aware mode. The v8 "snap-aliases" test
 * exercised the removed top-level `spwn snap` alias; snap now lives only at
 * `spwn world snap <subcmd>`, so these target the real command path.
 *
 * `spwn world snap save` takes a full world *id* (not a world key), read
 * off the container's label map via inspect and fed into a follow-up
 * result; the first run's container stays alive until its `await using`
 * scope exits. Snapshots are docker images that persist on the daemon
 * across runs, so an afterEach hook force-removes every `spwn-snapshot:*`
 * image. That cleanup is genuine test plumbing (image removal has no
 * framework accessor), so it keeps raw `node:child_process`.
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

    test('world snap --help lists the save/ls/restore/rm subcommands', async () => {
        // Given - the snap help rendered from any project
        await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('world snap --help');

        // Then - cobra renders the four subcommands on stdout (scalpel: third-party cobra help layout)
        expect(result.exitCode).toBe(0);
        expect(result.stdout).toContain('save');
        expect(result.stdout).toContain('ls');
        expect(result.stdout).toContain('restore');
        expect(result.stdout).toContain('rm');
    });

    test('world snap save + snap ls + snap rm round-trip', async () => {
        // Given - a running world whose runtime id is read off its container labels
        await using up = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');
        expect(up.exitCode).toBe(0);
        const neo = up.container('neo');
        expect(neo.running).toBe(true);
        const inspectData = neo.inspect.value as {
            Config?: { Labels?: Record<string, string> };
        };
        const worldId = inspectData.Config?.Labels?.['sh.spwn.world.id'];
        expect(worldId).toBeTruthy();
        expect(typeof worldId).toBe('string');

        // When - a snapshot is saved by id
        await using save = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(`world snap save ${worldId} --name round-trip`);

        // Then - the save banner is product-owned deterministic output: pin it byte-for-byte
        // (the snapshot tag's only dynamic segment is the world-id hash suffix — {{hex}})
        expect(save.exitCode).toBe(0);
        expect(save.stderr).toMatch('snap-saved.txt');

        // And the snapshot appears in the ls table
        await using list = await cli.fixture('$FIXTURES/docker-pilot/').exec('world snap ls');
        expect(list.exitCode).toBe(0);
        expect(list.stderr).toContain('round-trip');

        // And after rm the snapshot is absent from both streams
        await using rm = await cli
            .fixture('$FIXTURES/docker-pilot/')
            .exec(`world snap rm ${worldId}--round-trip`);
        expect(rm.exitCode).toBe(0);
        await using after = await cli.fixture('$FIXTURES/docker-pilot/').exec('world snap ls');
        expect(after.exitCode).toBe(0);
        expect(after.stderr).not.toContain('round-trip');
        expect(after.stdout).not.toContain('round-trip');
    });
});
