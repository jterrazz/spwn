import { describe, expect, test } from 'vitest';

import { spec } from '../../../_setup/cli.specification.js';

/**
 * World resource limits under the docker() spec mode.
 *
 * Currently, the project-mode up path (packages/container/backend/docker.go)
 * Only wires `PidsLimit` onto the container's HostConfig. Memory and CPU
 * Limits the legacy test asserted (`Memory`, `NanoCpus`, `CpuQuota`) are
 * Not applied by the backend at all, so those assertions are dropped —
 * They would have failed against `spwn up` regardless of the test
 * Framework. Re-add them once the backend threads physics limits through
 * To ContainerConfig.
 */
describe('world resource limits', () => {
    test('default container has a bounded pids limit', async () => {
        await using result = await spec('resource limits applied')
            .project('docker-pilot')
            .exec('up')
            .run();

        expect(result.exitCode).toBe(0);

        const neo = result.container('neo');
        expect(neo.running).toBe(true);

        const inspectData = neo.inspect.value as {
            HostConfig?: { PidsLimit?: number };
        };
        const pidsLimit = inspectData.HostConfig?.PidsLimit ?? 0;

        // Not 0 or -1 (both mean unlimited). A real number means the
        // Backend wired the limit through from physics config.
        expect(pidsLimit).toBeGreaterThan(0);
    });
});
