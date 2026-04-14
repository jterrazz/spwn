import { oxlint } from '@jterrazz/codestyle';
import { defineConfig } from 'oxlint';

export default defineConfig({
    extends: [oxlint.node],
});
