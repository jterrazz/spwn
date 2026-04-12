import { defineConfig, devices } from "@playwright/test";

/**
 * Full-stack UI E2E test configuration.
 *
 * Architecture:
 *   1. Global setup builds the Go binary and starts the API server
 *   2. Playwright webServer starts the Next.js dev server
 *   3. Tests run against the real UI + real API + real Docker
 *   4. Global teardown stops the API server and cleans up
 *
 * Requirements:
 *   - Docker running (for world spawning tests)
 *   - Go 1.25+ (binary is built fresh in global setup)
 *   - spwn-test:latest image built (`make build-test-image`)
 *   - At least one API key configured (for agent talk tests)
 */
export default defineConfig({
  testDir: "./specs",
  timeout: 60_000,
  expect: { timeout: 10_000 },
  fullyParallel: false, // worlds share Docker state — run sequentially
  retries: 0,
  workers: 1,
  reporter: [
    ["list"],
    ["html", { open: "never", outputFolder: "../playwright-report" }],
  ],

  use: {
    baseURL: "http://localhost:1420",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
    actionTimeout: 15_000,
  },

  // The Next.js dev server — started automatically by Playwright
  webServer: {
    command: "cd ../apps/observatory && NEXT_PUBLIC_API_URL=http://localhost:9877 npm run dev -- -p 1420",
    port: 1420,
    timeout: 30_000,
    reuseExistingServer: !process.env.CI,
    env: {
      NEXT_PUBLIC_API_URL: "http://localhost:9877",
    },
  },

  // The Go API server is started in global setup (not here, because
  // it needs a fresh binary build and SPWN_HOME isolation)
  globalSetup: "./setup/global-setup.ts",
  globalTeardown: "./setup/global-teardown.ts",

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
