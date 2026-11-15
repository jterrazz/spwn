import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    kind: 'website',
    platform: 'react',
    presets: [test()],
    typescript: {
        architecture: {
            // hazard: the hexagonal map derived from src/ refuses a spec that crosses its layers, so the layers below
            // hazard: hold the hexagon, each sparing the specs.
            hexagonal: false,
            layers: [
                {
                    name: 'ui-knows-no-app',
                    from: { path: '^src/presentation/ui/', pathNot: '/__tests__/' },
                    to: {
                        path: '^src/',
                        pathNot: ['^src/presentation/ui/', '^src/domain/(tone|separator)[.]ts$'],
                    },
                    why: 'the ui kit draws what it is handed and knows nothing of the engine, the navigation or a page',
                    fix: 'take the value as a prop, or move the code that needs the app out of src/presentation/ui/ (apps/desktop/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
                {
                    name: 'domain-imports-no-layer',
                    from: { path: '^src/domain/', pathNot: '/__tests__/' },
                    to: { path: '^src/', pathNot: '^src/domain/' },
                    why: 'the domain is plain functions and types over values: no word, no drawing, no link, and no other layer of the app',
                    fix: 'move the code that reads the engine to src/presentation/dependencies/, or the drawing to src/presentation/ui/ (apps/desktop/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
                {
                    name: 'use-cases-see-domain-and-ports',
                    from: { path: '^src/application/', pathNot: '/__tests__/' },
                    to: { path: '^src/', pathNot: '^src/(domain|application)/' },
                    why: 'a use case reads the engine, the opener and the presence through its ports alone, and draws nothing',
                    fix: 'name what the use case needs in a port of src/application/ports/ and let src/composition/ hand it the adapter (apps/desktop/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
                {
                    name: 'adapters-draw-nothing',
                    from: { path: '^src/infrastructure/', pathNot: '/__tests__/' },
                    to: { path: '^src/(presentation|pages)/' },
                    why: "an adapter speaks to the engine's socket, the Tauri shell or the document, and answers in the ports' types; the words and the drawing are the app's",
                    fix: "return the port's type and let the hooks of src/presentation/dependencies/ or a page word and draw it (apps/desktop/README.md, The dependency rule)",
                    read: 'docs/concepts/architecture.md#the-tree',
                },
                {
                    name: 'only-composition-builds-adapters',
                    from: {
                        path: '^src/',
                        pathNot: ['^src/composition/', '^src/infrastructure/', '/__tests__/'],
                    },
                    to: { path: '^src/infrastructure/' },
                    why: 'the adapters are built in one place, src/composition/compose.ts, and every other module reads them through a port',
                    fix: 'take the port from the provider (src/presentation/dependencies/dependencies.tsx), or build the adapter in src/composition/compose.ts (apps/desktop/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
                {
                    name: 'only-main-composes',
                    from: {
                        path: '^src/',
                        pathNot: ['^src/main[.]tsx$', '^src/composition/', '/__tests__/'],
                    },
                    to: { path: '^src/composition/' },
                    why: 'the window is composed once, by src/main.tsx, which hands the dependencies to the provider',
                    fix: 'read the dependencies through the provider, or take them as a prop from src/main.tsx (apps/desktop/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
                {
                    name: 'words-draw-nothing',
                    from: { path: '^src/presentation/words/', pathNot: '/__tests__/' },
                    to: { path: '^src/(presentation/(ui|navigation)|pages)/' },
                    why: "the words, tones and provisional reads name what the engine answered; how it is drawn is the pages' and the ui's",
                    fix: 'return a code, a word or a tone (src/domain/tone.ts) and let the caller draw it (apps/desktop/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
                {
                    name: 'dependencies-draw-nothing',
                    from: { path: '^src/presentation/dependencies/', pathNot: '/__tests__/' },
                    to: { path: '^src/(presentation/(ui|navigation)|pages)/' },
                    why: "the provider and its hooks hand the engine, the opener and the presence to a component; how an answer is drawn is the pages' and the ui's",
                    fix: 'return what the engine answered and let the caller draw it (apps/desktop/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
                {
                    name: 'navigation-knows-no-page',
                    from: { path: '^src/presentation/navigation/', pathNot: '/__tests__/' },
                    to: { path: '^src/pages/' },
                    why: 'the pages are handed to the navigation by src/pages/registry.tsx, so the navigation never imports one back',
                    fix: 'add the page to the registry, or move what the navigation needs out of src/pages/ (apps/desktop/README.md, The dependency rule)',
                    read: 'docs/concepts/architecture.md#the-tree',
                },
            ],
        },
        oxlint: {
            overrides: [
                {
                    files: ['src/presentation/ui/chart.tsx', 'src/presentation/ui/row.tsx'],
                    reason: 'a mark of a chart and a column of a row are their position: the list is never reordered and holds no other identity',
                    rules: { 'react/no-array-index-key': 'off' },
                },
                {
                    files: ['src/presentation/navigation/find.tsx'],
                    reason: 'the palette is a combobox: an option is chosen through the input that keeps the focus and names it by aria-activedescendant, so the option itself takes neither focus nor keys, and a click picks it',
                    rules: {
                        'jsx-a11y/click-events-have-key-events': 'off',
                        'jsx-a11y/interactive-supports-focus': 'off',
                    },
                },
                {
                    files: ['src/presentation/ui/code-block.tsx'],
                    reason: 'a code block, a log among them, is a scrolling region the keyboard must reach to scroll it, so the box takes the Tab focus',
                    rules: { 'jsx-a11y/no-noninteractive-tabindex': 'off' },
                },
                {
                    files: ['src/presentation/navigation/sidebar.tsx'],
                    reason: 'the sidebar is a navigation landmark whose buttons move by arrows: the landmark listens for the arrows its buttons bubble up',
                    rules: { 'jsx-a11y/no-noninteractive-element-interactions': 'off' },
                },
                {
                    files: ['scripts/capture.ts'],
                    reason: 'the capture script drives headless Chromium through Playwright, the browser the component specs run in, to photograph the proof pages',
                    rules: { 'jterrazz/no-test-code-in-production': 'off' },
                },
                {
                    files: ['src/presentation/words/words.ts'],
                    reason: "the law's one table of the words the engine owes, one literal map per code, so it grows with the wire and shrinks as each enum joins packages/api's words.json; a name of its own (`*.data.ts`) would move every import of it",
                    rules: { 'max-lines': 'off' },
                },
                {
                    files: ['src/presentation/ui/qr.ts'],
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
                    'src/presentation/ui/document.css loads the font by a CSS @import, which knip does not read',
                '@fontsource/geist-mono':
                    'src/presentation/ui/document.css loads the font by a CSS @import, which knip does not read',
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
