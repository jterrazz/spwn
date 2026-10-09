import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    kind: 'website',
    platform: 'next',
    presets: [test()],
    typescript: {
        oxlint: {
            overrides: [
                {
                    files: ['src/tauri/**'],
                    reason: 'the Tauri shell injects its bridge as globalThis.__TAURI__, a name this code reads and does not choose',
                    rules: { 'eslint/no-underscore-dangle': 'off' },
                },
            ],
        },
        roots: {
            'src-tauri':
                'the Tauri desktop shell: its CLI reads the Rust crate and tauri.conf.json from this folder by name',
        },
    },
});
