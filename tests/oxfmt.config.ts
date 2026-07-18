import { oxfmt } from '@jterrazz/typescript';
import { defineConfig } from 'oxfmt';

export default defineConfig({
    ...oxfmt,
    /*
     * Committed fixture trees and expected-output snapshots are byte-for-byte
     * significant — they lock in spwn's CLI output and YAML/project fixtures —
     * so the formatter must leave them alone.
     */
    ignorePatterns: [
        ...(oxfmt.ignorePatterns ?? []),
        'specs/fixtures/**',
        'specs/cli/**/expected/**',
        'specs/cli/**/fixtures/**',
        'web/**',
        '_smoke/**',
        '_catalog/**',
        '_contracts/**',
        '_simulators/**',
    ],
});
