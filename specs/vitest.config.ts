import { defineSpecConfig } from '@jterrazz/test/vitest';

export default defineSpecConfig({
    /*
     * A `<case>.spec.yaml` beside a spec IS a test file: the plugin adds the
     * glob to the include and transforms each document into a one-test module
     * bound to the runner named here. The runner then reports the document's
     * own path and its `description:`, so a failing scenario opens where it is
     * written. Declared as a key rather than a plugin because this config has
     * no `projects` to attach it to.
     */
    literate: {
        include: ['cli/**/*.spec.yaml'],
        specification: './cli/cli.specification.ts',
    },
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
        /*
         * Under `specs/`, a chain that meets the assembled product is a
         * `.spec.ts` (C12). Two files keep `.test.ts` because they reach no
         * runner (C18, recorded): `cli/logs/api.test.ts`, a payload-shape
         * assertion that spawns nothing. `lint/` is not a facet — it is a
         * repository suite, and its word stays `.test.ts`.
         */
        include: ['cli/**/*.spec.ts', 'cli/**/*.test.ts', 'lint/**/*.test.ts'],
        /*
         * Additive: the preset already excludes vitest's defaults and every
         * `_fixtures/` tree. Left to state is the playwright suite, and the
         * `cli/smoke/` domain, whose specs rebuild a world image from scratch
         * (~minutes) and run via vitest.smoke.config.ts.
         */
        exclude: ['web/**', 'cli/smoke/**'],
    },
});
