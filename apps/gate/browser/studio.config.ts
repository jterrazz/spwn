import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

export default defineConfig({
    kind: 'api',
    platform: 'node',
    presets: [test()],
    typescript: {
        oxlint: {
            overrides: [
                {
                    files: ['src/**'],
                    reason: 'the sidecar IS a browser driver: playwright is its production dependency, the Chromium every catalog tool browses through, not test material',
                    rules: { 'jterrazz/no-test-code-in-production': 'off' },
                },
            ],
        },
    },
});
