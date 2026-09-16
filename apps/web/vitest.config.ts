import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';

export default defineConfig({
    resolve: {
        alias: {
            /* The `@/*` paths of tsconfig.json, which vitest does not read. */
            '@': fileURLToPath(new URL('src', import.meta.url)),
        },
    },
    test: {
        environment: 'node',
        include: ['src/**/*.test.ts', 'src/**/*.test.tsx'],
        testTimeout: 10_000,
    },
});
