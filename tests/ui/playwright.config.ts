import { defineConfig, devices } from '@playwright/test';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(__dirname, '../..');
const BIN = resolve(REPO_ROOT, 'bin/spwn');

/**
 * Full-stack UI E2E tests.
 *
 * Both servers (Go API + Next.js) are managed by Playwright's
 * webServer - they start before tests and stop after. No global
 * setup/teardown needed for server lifecycle.
 *
 * Uses the real ~/.spwn state - these are true integration tests.
 */
export default defineConfig({
    testDir: './specs',
    timeout: 60_000,
    expect: { timeout: 10_000 },
    fullyParallel: false,
    retries: 0,
    workers: 1,
    reporter: [['list'], ['html', { open: 'never', outputFolder: '../playwright-report' }]],

    use: {
        baseURL: 'http://localhost:1420',
        trace: 'retain-on-failure',
        screenshot: 'only-on-failure',
        video: 'retain-on-failure',
        actionTimeout: 15_000,
    },

    webServer: [
        {
            command: `${BIN} dash start --port 9877`,
            port: 9877,
            timeout: 15_000,
            reuseExistingServer: !process.env.CI,
        },
        {
            command: 'npx next dev -p 1420',
            cwd: resolve(REPO_ROOT, 'apps/web'),
            port: 1420,
            timeout: 30_000,
            reuseExistingServer: !process.env.CI,
            env: {
                ...process.env,
                NEXT_PUBLIC_API_URL: 'http://localhost:9877',
            },
        },
    ],

    globalSetup: './setup/global-setup.ts',
    globalTeardown: './setup/global-teardown.ts',

    projects: [
        {
            name: 'chromium',
            use: { ...devices['Desktop Chrome'] },
        },
    ],
});
