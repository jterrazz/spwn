import { base, defineConfig } from '@jterrazz/typescript/oxfmt';

export default defineConfig({
    ...base,
    ignorePatterns: [
        ...(base.ignorePatterns ?? []),
        // reason: a literate spec carries the CLI's expected streams inside a
        // `stdout: |+` block, where a trailing space IS output — the formatter
        // strips it and silently rewrites the assertion. The document's shape is
        // checked by the @jterrazz/test conventions checker instead.
        'specs/cli/**/*.spec.yaml',
        // reason: Go's own golden convention — `testdata/` holds the catalog
        // tests' byte-for-byte inputs and expected bundles, compared verbatim.
        '_catalog/testdata/**',
    ],
});
