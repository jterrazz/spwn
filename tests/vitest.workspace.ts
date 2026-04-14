import { defineWorkspace } from "vitest/config";

export default defineWorkspace([
  {
    test: {
      name: "cli",
      include: [
        "e2e/cli/**/*.e2e.test.ts",
        "e2e/init/**/*.e2e.test.ts",
        "e2e/errors/**/*.e2e.test.ts",
        "e2e/marketplace/**/*.e2e.test.ts",
        "e2e/status/**/*.e2e.test.ts",
        "e2e/system/**/*.e2e.test.ts",
      ],
    },
  },
  {
    test: {
      name: "docker",
      fileParallelism: false,
      include: [
        "e2e/world/**/*.e2e.test.ts",
        "e2e/agent/**/*.e2e.test.ts",
        "e2e/colony/**/*.e2e.test.ts",
        "e2e/config/**/*.e2e.test.ts",
        "e2e/state/**/*.e2e.test.ts",
        "e2e/messaging/**/*.e2e.test.ts",
        "e2e/lifecycle/**/*.e2e.test.ts",
      ],
    },
  },
]);
