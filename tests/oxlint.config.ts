import { oxlint } from '@jterrazz/codestyle';
import { defineConfig } from 'oxlint';

export default defineConfig({
    extends: [oxlint.node],
    rules: {
        // Tests/setup files mix exported types and helpers with private
        // Helpers; reorganising every file just to satisfy exports-last
        // Hurts readability without buying anything.
        'import/exports-last': 'off',
    },
});
