import { testing } from '@jterrazz/test/oxlint';
import { compose, defineConfig, next } from '@jterrazz/typescript/oxlint';

export default defineConfig(compose(next, testing));
