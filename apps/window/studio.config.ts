import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    kind: 'website',
    platform: 'react',
    presets: [test()],
    typescript: {
        architecture: {
            layers: [
                {
                    name: 'design-knows-no-app',
                    from: { path: '^src/design/', pathNot: '/__tests__/' },
                    to: {
                        path: '^src/',
                        pathNot: ['^src/design/', '^src/tone[.]ts$', '^src/separator[.]ts$'],
                    },
                    why: 'the design kit draws what it is handed and knows nothing of the engine, the shell or a page',
                    fix: 'take the value as a prop, or move the code that needs the app out of src/design/ (apps/window/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
                {
                    name: 'model-is-pure',
                    from: { path: '^src/model/', pathNot: '/__tests__/' },
                    to: { path: '^src/', pathNot: ['^src/model/', '^src/separator[.]ts$'] },
                    why: 'the model is plain functions over values: no word, no tone, no drawing, no link',
                    fix: 'move the code that reads the engine to src/engine/, or the drawing to src/design/ (apps/window/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
                {
                    name: 'engine-draws-nothing',
                    from: { path: '^src/engine/', pathNot: '/__tests__/' },
                    to: { path: '^src/(design|shell|pages)/' },
                    why: "the engine layer reads the link and names what it answered; how it is drawn is the pages' and the shell's",
                    fix: 'return a code, a word or a tone (src/tone.ts) and let the caller draw it (apps/window/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
                {
                    name: 'shell-knows-no-page',
                    from: { path: '^src/shell/', pathNot: '/__tests__/' },
                    to: { path: '^src/pages/' },
                    why: 'the pages are handed to the shell by src/pages/registry.tsx, so the shell never imports one back',
                    fix: 'add the page to the registry, or move what the shell needs out of src/pages/ (apps/window/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
            ],
        },
        oxlint: {
            overrides: [
                {
                    files: ['src/design/chart.tsx', 'src/design/row.tsx'],
                    reason: 'a mark of a chart and a column of a row are their position: the list is never reordered and holds no other identity',
                    rules: { 'react/no-array-index-key': 'off' },
                },
                {
                    files: ['src/shell/find.tsx'],
                    reason: 'the palette is a combobox: an option is chosen through the input that keeps the focus and names it by aria-activedescendant, so the option itself takes neither focus nor keys, and a click picks it',
                    rules: {
                        'jsx-a11y/click-events-have-key-events': 'off',
                        'jsx-a11y/interactive-supports-focus': 'off',
                    },
                },
                {
                    files: ['src/design/code-block.tsx'],
                    reason: 'a code block, a log among them, is a scrolling region the keyboard must reach to scroll it, so the box takes the Tab focus',
                    rules: { 'jsx-a11y/no-noninteractive-tabindex': 'off' },
                },
                {
                    files: ['src/shell/sidebar.tsx'],
                    reason: 'the sidebar is a navigation landmark whose buttons move by arrows: the landmark listens for the arrows its buttons bubble up',
                    rules: { 'jsx-a11y/no-noninteractive-element-interactions': 'off' },
                },
                {
                    files: ['scripts/capture.ts'],
                    reason: 'the capture script drives headless Chromium through Playwright, the browser the component specs run in, to photograph the proof pages',
                    rules: { 'jterrazz/no-test-code-in-production': 'off' },
                },
                {
                    files: ['src/engine/words.ts'],
                    reason: "the law's one table of the words the engine owes, one literal map per code, so it grows with the wire and shrinks as each enum joins packages/api's words.json; a name of its own (`*.data.ts`) would move every import of it",
                    rules: { 'max-lines': 'off' },
                },
                {
                    files: ['src/design/qr.ts'],
                    reason: 'a QR code is bits: its Reed-Solomon field multiplies by shifts and XOR, and its codewords are read bit by bit',
                    rules: { 'no-bitwise': 'off' },
                },
            ],
        },
        oxfmt: {
            ignore: {
                'src/fixtures/worlds/**':
                    "written by scripts/record.py from the engine's real answers, byte for byte what it recorded",
            },
        },
        knip: {
            entry: {
                'scripts/capture.ts':
                    'run by node in the Linux VM to photograph the proof pages; nothing imports it',
            },
            ignoreDependencies: {
                '@fontsource/geist':
                    'src/design/document.css loads the font by a CSS @import, which knip does not read',
                '@fontsource/geist-mono':
                    'src/design/document.css loads the font by a CSS @import, which knip does not read',
            },
        },
        roots: {
            scripts:
                "the capture scripts that photograph the built pages in headless Chromium and the Tauri window under Xvfb, and the recorder of the engine's answers the replay transport serves, in a Linux VM only",
            'src-tauri':
                'the Tauri desktop shell: its CLI reads the Rust crate and tauri.conf.json from this folder by name',
        },
    },
});
