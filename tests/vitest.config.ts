import { defineConfig } from 'vitest/config';

export default defineConfig({
    test: {
        testTimeout: 120_000, // 2 minutes per test (Docker is slow)
        hookTimeout: 60_000,
        fileParallelism: false, // Docker tests must be sequential
        // UI specs live under tests/ui/ and run via Playwright
        // (`npm run test:ui`). Keep vitest away from them.
        exclude: ['**/node_modules/**', '**/dist/**', 'ui/**'],
        projects: [
            {
                test: {
                    name: 'cli',
                    testTimeout: 120_000,
                    hookTimeout: 60_000,
                    include: [
                        'e2e/cli/**/*.e2e.test.ts',
                        'e2e/init/**/*.e2e.test.ts',
                        'e2e/errors/**/*.e2e.test.ts',
                        'e2e/marketplace/**/*.e2e.test.ts',
                        'e2e/status/**/*.e2e.test.ts',
                        'e2e/system/**/*.e2e.test.ts',
                        'e2e/logs/**/*.e2e.test.ts',
                        'e2e/agent/crud/**/*.e2e.test.ts',
                        'e2e/agent/export/**/*.e2e.test.ts',
                        'e2e/agent/evolution/**/*.e2e.test.ts',
                        'e2e/web/web/**/*.e2e.test.ts',
                    ],
                },
            },
            {
                test: {
                    name: 'docker',
                    fileParallelism: false,
                    testTimeout: 120_000,
                    hookTimeout: 60_000,
                    include: [
                        'e2e/world/**/*.e2e.test.ts',
                        'e2e/agent/**/*.e2e.test.ts',
                        'e2e/colony/**/*.e2e.test.ts',
                        'e2e/config/**/*.e2e.test.ts',
                        'e2e/state/**/*.e2e.test.ts',
                        'e2e/messaging/**/*.e2e.test.ts',
                        'e2e/lifecycle/**/*.e2e.test.ts',
                        'e2e/architect/**/*.e2e.test.ts',
                        'e2e/knowledge/**/*.e2e.test.ts',
                    ],
                    exclude: [
                        '**/node_modules/**',
                        '**/dist/**',
                        'e2e/agent/crud/**',
                        'e2e/agent/export/**',
                        'e2e/agent/evolution/**',
                    ],
                },
            },
        ],
    },
});
