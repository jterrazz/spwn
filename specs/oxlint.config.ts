import { testing } from '@jterrazz/test/oxlint';
import { compose, defineConfig, node } from '@jterrazz/typescript/oxlint';

/*
 * Node preset + the @jterrazz/test conventions plugin. Every tree this member
 * owns is source — the CLI specs, the Playwright suite, the smoke test, the
 * contract asserter. What is not source, `_fixtures/` inputs and `_expected/`
 * goldens, the profile already ignores; the Go catalog tests and the shell
 * simulators hold nothing oxlint reads.
 */
export default defineConfig(
    compose(node, testing, {
        overrides: [
            {
                files: ['vitest.config.ts', 'vitest.smoke.config.ts'],
                /*
                 * reason: this package's root IS the specs root, so F3 reads its
                 * vitest configs as specs deep-importing the framework. They are
                 * tool files, and `@jterrazz/test/vitest` is the published entry
                 * a vitest config is meant to import — the one deep import F3
                 * exempts is `@jterrazz/test/oxlint`.
                 */
                rules: { 'jterrazz/f3-specs-public-entry': 'off' },
            },
        ],
    }),
);
