import { testing } from '@jterrazz/test/oxlint';
import { compose, node } from '@jterrazz/typescript/oxlint';
import { defineConfig } from 'oxlint';

/*
 * Node preset + the @jterrazz/test conventions plugin (the 38+ rule
 * catalogue). Fixtures and expected-output snapshots are committed
 * verbatim and must not be linted.
 */
export default defineConfig(
    compose(node, testing, {
        ignorePatterns: [
            'node_modules',
            'dist',
            'web',
            '_smoke',
            '_catalog',
            '_contracts',
            '_simulators',
            'specs/fixtures/**',
            'specs/cli/**/expected/**',
            'specs/cli/**/fixtures/**',
        ],
    }),
);
