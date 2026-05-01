import { defineConfig } from 'vitest/config';

// Smoke tests: real-build end-to-end coverage for the default scaffold
// And every shipped catalog example. These intentionally bypass
// SPWN_BASE_IMAGE so the real image build + tool probe path runs,
// Which is the only way to catch "my scaffold generates a broken
// World" regressions before they reach users.
//
// Run with: pnpm test:smoke
export default defineConfig({
    test: {
        exclude: ['**/node_modules/**', '**/dist/**', 'web/**'],
        include: ['_smoke/**/*.e2e.test.ts'],
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
