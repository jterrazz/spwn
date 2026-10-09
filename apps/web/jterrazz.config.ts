import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    kind: 'website',
    platform: 'next',
    presets: [test()],
    typescript: {
        roots: {
            'src-tauri':
                'the Tauri desktop shell: its CLI reads the Rust crate and tauri.conf.json from this folder by name',
        },
    },
});
