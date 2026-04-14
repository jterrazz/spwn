import { oxlint } from '@jterrazz/codestyle';
import { defineConfig } from 'oxlint';

export default defineConfig({
    extends: [oxlint.next],
    rules: {
        // Next.js page components commonly intersperse helper consts with
        // The default export; requiring exports-last would force large
        // Reorderings that hurt readability.
        'import/exports-last': 'off',
        // Shadcn-generated ui components use `import * as React`; converting
        // Them to named imports would make shadcn sync noisier than it's worth.
        'import/no-namespace': 'off',
    },
});
