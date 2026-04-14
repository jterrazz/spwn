import { defineConfig } from 'vitest/config';

export default defineConfig({
    test: {
        exclude: ['**/node_modules/**', '**/dist/**', 'ui/**'],
        // 2 minutes per test because the Docker-asserting tests spawn
        // Real containers; CLI-only tests finish in milliseconds so the
        // Upper bound is harmless for them.
        testTimeout: 120_000,
        hookTimeout: 60_000,
        // Serial file execution: spwn's command routing (msg, down,
        // Destroy, inspect) still looks up containers daemon-wide by
        // `sh.spwn.world.config` name, not by the per-test-run label —
        // So two parallel tests both spawning a `neo` world would step
        // On each other at CLI-dispatch time. Until spwn honours
        // SPWN_TEST_LABEL on the routing side too, keep tests serial.
        // The framework's per-test cleanup still makes the suite safe
        // Across runs even though runs themselves are sequential.
        fileParallelism: false,
        include: ['e2e/**/*.e2e.test.ts'],
    },
});
