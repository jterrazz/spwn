import { testing } from '@jterrazz/test/oxlint';
import { compose, defineConfig, node } from '@jterrazz/typescript/oxlint';

/*
 * Node preset + the @jterrazz/test conventions plugin. The vitest trees this
 * member owns are all source — the CLI specs, the smoke test, the contract
 * asserter. What is not source, `_fixtures/` inputs and `_expected/` goldens,
 * the profile already ignores; the Go catalog tests and the shell simulators
 * hold nothing oxlint reads.
 */
export default defineConfig(
    compose(node, testing, {
        ignorePatterns: [
            /*
             * reason: `tests/web/` is a Playwright suite, not a vitest one, and its
             * files carry the same `*.spec.ts` name. The rulebook reads that name as
             * a vitest spec: `vitest/prefer-importing-vitest-globals` auto-fixes each
             * file with `import { expect, test } from 'vitest'`, shadowing the
             * Playwright fixtures the file imports and breaking the suite at run time.
             */
            'web/**',
        ],
    }),
);
