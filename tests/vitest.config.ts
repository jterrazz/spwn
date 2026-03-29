import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    testTimeout: 120_000, // 2 minutes per test (Docker is slow)
    hookTimeout: 60_000,
    fileParallelism: false, // Docker tests must be sequential
  },
});
