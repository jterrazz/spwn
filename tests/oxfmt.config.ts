import { oxfmt } from '@jterrazz/codestyle';
import { defineConfig } from 'oxfmt';

export default defineConfig({
    ...oxfmt,
    // Committed fixture content and expected-output snapshots are
    // Byte-for-byte significant — they lock in spwn's CLI output,
    // YAML fixtures, seed files — so the formatter must leave them
    // Alone. (`.gitignore` already handles generated artefacts
    // Like spwn/, .spwn/, playwright-report/, test-results/.)
    ignorePatterns: [
        ...(oxfmt.ignorePatterns ?? []),
        'fixtures/**',
        'cli/**/expected/**',
        'cli/**/seeds/**',
        'catalog/testdata/**',
    ],
});
