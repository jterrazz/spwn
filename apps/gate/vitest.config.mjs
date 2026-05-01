import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['sdk/**/*.test.mjs'],
    testTimeout: 10_000,
  },
});
