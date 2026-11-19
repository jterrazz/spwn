import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    presets: [test()],
    typescript: {
        oxfmt: {
            ignore: {
                'packages/**/__tests__/__fixtures__/**':
                    'the input trees of the Go tests and expected outputs, compared byte for byte with what the product writes: a fixture reformatted or re-commented would no longer be the one the product reads or emits',
                'catalog/**':
                    'the built-in `spwn:*` catalog, embedded into the binary and copied verbatim into the project of a user: its YAML comments and Markdown are the documentation that user reads, kept as authored',
                'specs/cli/**/*.spec.yaml':
                    'the streams of a document are byte-exact, and the document formatter writes the blank first line of a `|+` stream (the help of cobra opens with one) as its indentation, which oxfmt strips as trailing whitespace: the two would rewrite each other forever',
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
            ignore: {
                'machines/**':
                    "station's tree, whose npm workspace knip cannot see from the root (pnpm's globs do not reach it): its configs, declarations and imports of its own packages read as unused or unlisted here, and its own `make check` runs knip over it with the right workspace",
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
            machines: {
                reason: 'station, imported with its history: a repository of its own with its own toolchain, gates and `make check`, run from inside it (`make machines-check`), so no root pass reads or rewrites it',
                verbatim: true,
            },
            docker: 'the build context of the test image: `make test-image` builds spwn-test:latest, the mock Claude and Codex runtimes baked in, from docker/spwn-test',
            scripts:
                'the installer `make install-binary` runs, the manual-pass harness, and the signing and notarization steps of the Publish workflow',
        },
    },
});
