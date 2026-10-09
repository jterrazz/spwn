import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    kind: 'workspace',
    presets: [test()],
    typescript: {
        generated: {
            catalog:
                'the built-in `spwn:*` catalog, embedded into the binary and copied verbatim into a user’s project: its YAML comments and Markdown are that user’s documentation, kept as authored',
        },
        oxfmt: {
            ignore: {
                'specs/cli/**/*.spec.yaml':
                    'a document’s streams are byte-exact, and the document formatter writes the blank first line of a `|+` stream (cobra’s help opens with one) as its indentation, which oxfmt strips as trailing whitespace — the two would rewrite each other forever',
                '**/testdata/**':
                    'Go keeps its suites’ inputs and goldens in testdata/, which the go tool never builds and the Go tests compare byte for byte',
            },
        },
        roots: {
            scripts: 'the installer `make install-binary` runs, and the manual-pass harness',
        },
    },
});
