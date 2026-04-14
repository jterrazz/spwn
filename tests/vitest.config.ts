import { defineConfig } from 'vitest/config';

export default defineConfig({
    test: {
        exclude: ['**/node_modules/**', '**/dist/**', 'web/**'],
        // 2 minutes per test because the Docker-asserting tests spawn
        // Real containers; CLI-only tests finish in milliseconds so the
        // Upper bound is harmless for them.
        testTimeout: 120_000,
        hookTimeout: 60_000,
        // Parallel file execution is safe: spwn's state.Store.List /
        // Get scope every world-lookup by the SPWN_TEST_LABEL env var
        // The framework injects per test run (see packages/world/
        // Internal/state/state.go), so two parallel tests both
        // Spawning a "neo" world route to their own container.
        // Combined with the framework's label-based cleanup on
        // Symbol.asyncDispose, the whole suite is isolated per file.
        fileParallelism: true,
        include: ['cli/**/*.e2e.test.ts'],
    },
});
