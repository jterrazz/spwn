import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    kind: 'workspace',
    presets: [test()],
    typescript: {
        oxfmt: {
            ignore: {
                'catalog/**':
                    'the built-in `spwn:*` catalog, embedded into the binary and copied verbatim into a user’s project: its YAML comments and Markdown are that user’s documentation, kept as authored',
                'specs/cli/**/*.spec.yaml':
                    'a document’s streams are byte-exact, and the document formatter writes the blank first line of a `|+` stream (cobra’s help opens with one) as its indentation, which oxfmt strips as trailing whitespace — the two would rewrite each other forever',
                '**/testdata/**':
                    'Go keeps its suites’ inputs and goldens in testdata/, which the go tool never builds and the Go tests compare byte for byte',
            },
        },
        knip: {
            entry: {
                'apps/spwn-cookie-sync/background.js':
                    'the extension service worker, which Chrome loads by the path manifest.json names',
                'apps/spwn-cookie-sync/popup.js':
                    'the extension popup script, which Chrome loads from popup.html',
                'catalog/x/tools/x/index.js':
                    'the X tool the gate runs by the path its tool.yaml names',
            },
            ignoreBinaries: {
                pgrep: 'a system binary of macOS and Linux, never a package',
                playwright:
                    'CI runs it through `pnpm -C specs exec` to install the headless browser, from the specs member that declares it',
            },
            ignoreDependencies: {
                turbo: 'the Makefile runs it as `pnpm exec turbo`; no source imports it',
                playwright:
                    'the optional peer @jterrazz/test imports at run time when a website specification visits a page',
            },
        },
        roots: {
            catalog:
                'the built-in `spwn:*` catalog the binary embeds: worlds, agents, skills and tools a user installs by name',
            docker: 'the test image’s build context: `make test-image` builds spwn-test:latest, the mock Claude and Codex runtimes baked in, from docker/spwn-test',
            scripts: 'the installer `make install-binary` runs, and the manual-pass harness',
        },
    },
});
