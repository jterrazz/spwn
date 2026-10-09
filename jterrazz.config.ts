import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    kind: 'workspace',
    presets: [test()],
    typescript: {
        roots: {
            catalog: 'the built-in `spwn:*` dependency catalog, embedded into the binary by go:generate',
            scripts: 'the installer `make install` runs',
        },
    },
});
