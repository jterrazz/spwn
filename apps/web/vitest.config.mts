import { defineSpecConfig, unit } from '@jterrazz/test/vitest';
import { fileURLToPath } from 'node:url';

/*
 * `.mts`, not `.ts`: this member declares no `"type": "module"`, so vite
 * bundles a `vitest.config.ts` as CJS, and CJS cannot `require()` the
 * ESM-only `@jterrazz/test/vitest`.
 */
export default defineSpecConfig({
    resolve: {
        alias: {
            /* The `@/*` paths of tsconfig.json, which vitest does not read. */
            '@': fileURLToPath(new URL('src', import.meta.url)),
        },
    },
    test: {
        projects: [unit({ roots: ['src'], timeout: 10_000 })],
    },
});
