import { defineConfig } from 'vitest/config';

export default defineConfig({
    test: {
        /*
         * 2 minutes per test: the docker-aware specs spawn real containers.
         * CLI-only specs finish in milliseconds, so the upper bound is harmless.
         */
        testTimeout: 120_000,
        /*
         * Setup hooks and the first container boot are slow on cold CI runners;
         * 180s gives headroom without hiding real regressions.
         */
        hookTimeout: 180_000,
        /*
         * Parallel file execution is safe: spwn scopes every world-lookup by the
         * SPWN_TEST_LABEL the framework injects per spec, and the framework
         * force-removes each run's containers by that label on Symbol.asyncDispose,
         * so parallel specs both spawning a "neo" world route to their own container.
         */
        fileParallelism: true,
        include: ['specs/cli/**/*.test.ts', 'specs/lint/**/*.test.ts'],
        /*
         * The real-build smoke test (specs/cli/smoke/init-up.test.ts) rebuilds a
         * world image from scratch (~minutes) and runs via vitest.smoke.config.ts.
         */
        exclude: [
            '**/node_modules/**',
            '**/dist/**',
            'web/**',
            '**/fixtures/**',
            'specs/cli/smoke/init-up.test.ts',
        ],
    },
});
