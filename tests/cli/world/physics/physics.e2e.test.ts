import type { CliResult } from '@jterrazz/test';
import { afterAll, beforeAll, describe, expect, test } from 'vitest';

import { spec } from '../../../setup/cli.specification.js';

/**
 * World physics.
 *
 * Every test here is a read-only assertion on a freshly-upped world,
 * so we share ONE container across the file via beforeAll/afterAll.
 * Spawning once instead of per-test cuts ~4× container create/start
 * cost out of the run. The framework's test-run label isolates this
 * shared container from any other file running in parallel.
 *
 * Legacy coverage dropped:
 *   - "inspect shows physics constants" — `spwn world inspect` no
 *     longer surfaces a Constants block; the feature is gone.
 *   - "physics.md contains CPU/Memory/Timeout" — physics.md template
 *     does not materialise constants anymore.
 */
describe('world physics', () => {
    let world: CliResult;

    beforeAll(async () => {
        world = await spec('world physics shared').project('docker-pilot').exec('up').run();
        expect(world.exitCode).toBe(0);
    });

    afterAll(async () => {
        await world[Symbol.asyncDispose]();
    });

    test('CLAUDE.md inlines physics and faculties', () => {
        const neo = world.container('neo');
        expect(neo.running).toBe(true);

        // Physics and faculties used to live as separate /world/*.md
        // Files. They're now inlined into every agent's CLAUDE.md so
        // The system prompt is self-contained at boot.
        const claude = neo.file('/agents/neo/CLAUDE.md').content;
        expect(claude).toMatch(/## Physics/);
        expect(claude).toMatch(/## Faculties/);
    });

    test('inlined physics documents the network law and topology', () => {
        const claude = world.container('neo').file('/agents/neo/CLAUDE.md').content;
        expect(claude).toMatch(/network/i);
        expect(claude).toMatch(/Laws/);
        expect(claude).toMatch(/\/workspace/);
    });

    test('inlined faculties lists available tools', () => {
        const claude = world.container('neo').file('/agents/neo/CLAUDE.md').content;
        expect(claude).toMatch(/spwn:unix/);
    });

    test('default network mode is bridge', () => {
        const inspectData = world.container('neo').inspect.value as {
            HostConfig?: { NetworkMode?: string };
        };
        expect(inspectData.HostConfig?.NetworkMode).toBe('bridge');
    });
});
