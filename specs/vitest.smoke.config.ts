import { defineSpecConfig } from '@jterrazz/test/vitest';

// Smoke tests: real-build end-to-end coverage for the default scaffold
// And every shipped catalog example. These intentionally bypass
// SPWN_BASE_IMAGE so the real image build + tool probe path runs,
// Which is the only way to catch "my scaffold generates a broken
// World" regressions before they reach users.
//
// Run with: pnpm test:smoke
export default defineSpecConfig({
    test: {
        /*
         * Additive on the preset's exclusions; only the playwright suite is
         * left to state.
         */
        exclude: ['web/**'],
        // Both real-build smokes are specs of the cli facet's `smoke` domain:
        // the framework-based scaffold smoke, and the raw-execSync upgrade one.
        // `upgrade.e2e.test.ts` keeps `.test.ts`: it drives the binary with raw
        // execSync and reaches no runner (C18, recorded).
        include: ['cli/smoke/**/*.spec.ts', 'cli/smoke/**/*.test.ts'],
        // Each test builds a world image from scratch on a cold run.
        // First apt-get in a fresh layer can easily take 3-5 minutes.
        testTimeout: 600_000,
        hookTimeout: 60_000,
        // Serialize file execution: every test writes to the shared
        // Spwn/world:latest tag. Parallel builds would race and thrash
        // The tag, producing nondeterministic results.
        fileParallelism: false,
    },
});
