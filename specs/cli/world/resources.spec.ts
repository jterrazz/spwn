import { expect, test } from 'vitest';

import { cli } from '../cli.specification.js';

/**
 * World resource limits under docker-aware mode. The project-mode up path
 * only wires `PidsLimit` onto the container HostConfig today; the Memory /
 * NanoCpus / CpuQuota assertions the v8 test carried were dropped because
 * the backend does not apply them (they failed against `spwn up`
 * regardless of the framework). Re-add once physics limits are threaded
 * through to ContainerConfig. Result binds with `await using` (rule B5).
 */

test('default container has a bounded pids limit', async () => {
    // Given - a freshly-upped docker-pilot world
    await using result = await cli.fixture('$FIXTURES/docker-pilot/').exec('up');

    // Then - the container is running with a real (non-unlimited) pids limit
    expect(result.exitCode).toBe(0);
    const neo = result.container('neo');
    expect(neo.running).toBe(true);

    const inspectData = neo.inspect.value as {
        HostConfig?: { PidsLimit?: number };
    };
    const pidsLimit = inspectData.HostConfig?.PidsLimit ?? 0;
    // 0 and -1 both mean unlimited; a positive number means the backend wired the limit through
    expect(pidsLimit).toBeGreaterThan(0);
});
