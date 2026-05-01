import { defineConfig, devices } from '@playwright/test';
import { cpSync, existsSync, mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(__dirname, '../..');
const BIN = resolve(REPO_ROOT, 'bin/spwn');
const SPWN_HOME = process.env.SPWN_WEB_E2E_HOME ?? mkdtempSync(resolve(tmpdir(), 'spwn-web-e2e-'));
const SPWN_PROJECT =
    process.env.SPWN_WEB_E2E_PROJECT ?? mkdtempSync(resolve(tmpdir(), 'spwn-web-e2e-project-'));
const SPWN_TEST_LABEL =
    process.env.SPWN_WEB_E2E_LABEL ??
    `web-e2e-${Date.now()}-${Math.random().toString(16).slice(2)}`;
const SPWN_TEST_CONFIG = resolve(SPWN_HOME, 'playwright-test-config.json');

function copyDir(src: string, dst: string) {
    if (!existsSync(src)) {
        throw new Error(`missing web e2e fixture source: ${src}`);
    }
    cpSync(src, dst, { force: true, recursive: true });
}

function prepareProject(root: string) {
    if (existsSync(join(root, 'spwn.yaml'))) {
        return;
    }
    mkdirSync(join(root, 'spwn', 'agents'), { recursive: true });
    mkdirSync(join(root, 'spwn', 'skills'), { recursive: true });
    mkdirSync(join(root, 'spwn', 'knowledge'), { recursive: true });

    copyDir(
        join(REPO_ROOT, 'catalog', 'matrix', 'agents', 'neo'),
        join(root, 'spwn', 'agents', 'neo'),
    );
    for (const agent of ['ceo', 'devops', 'analyst']) {
        copyDir(
            join(REPO_ROOT, 'catalog', 'startup', 'agents', agent),
            join(root, 'spwn', 'agents', agent),
        );
    }
    for (const skill of [
        ['matrix', 'world-exploration.md'],
        ['matrix', 'self-reflection.md'],
        ['startup', 'code-review.md'],
        ['startup', 'deployment.md'],
        ['startup', 'sprint-planning.md'],
    ] as const) {
        copyDir(
            join(REPO_ROOT, 'catalog', skill[0], 'skills', skill[1]),
            join(root, 'spwn', 'skills', skill[1]),
        );
    }

    writeFileSync(
        join(root, 'spwn.yaml'),
        `version: 1
name: web-e2e

dependencies:
  - "spwn:unix"
  - "spwn:git"

worlds:
  matrix:
    agents: [neo]
    workspaces: [.]
    knowledge: ./spwn/knowledge
  startup:
    agents: [ceo, devops, analyst]
    workspaces: [.]
    knowledge: ./spwn/knowledge
`,
    );
    writeFileSync(join(root, '.gitignore'), '.spwn/\n');
}

prepareProject(SPWN_PROJECT);
// Skip the first-run onboarding gate so the UI lands on the actual pages under test instead of the welcome wizard.
mkdirSync(SPWN_HOME, { recursive: true });
writeFileSync(resolve(SPWN_HOME, '.onboarding-complete'), '');
writeFileSync(
    SPWN_TEST_CONFIG,
    JSON.stringify(
        { spwnHome: SPWN_HOME, spwnProject: SPWN_PROJECT, testLabel: SPWN_TEST_LABEL },
        null,
        2,
    ),
);

const testEnv: Record<string, string> = {
    ...Object.fromEntries(
        Object.entries(process.env).filter(
            (entry): entry is [string, string] => typeof entry[1] === 'string',
        ),
    ),
    SPWN_HOME,
    SPWN_TEST_LABEL,
    SPWN_SKIP_AUTH_VALIDATION: '1',
    SPWN_TEST_CONFIG,
};
if (!process.env.SPWNE2E_REAL_IMAGE) {
    testEnv.SPWN_BASE_IMAGE = 'spwn-test:latest';
}

/**
 * Full-stack UI E2E tests.
 *
 * Both servers (Go API + Next.js) are managed by Playwright's
 * webServer - they start before tests and stop after. No global
 * setup/teardown needed for server lifecycle.
 *
 * Uses an isolated SPWN_HOME and test-run Docker label so tests never
 * touch the developer's real ~/.spwn or unrelated containers.
 */
export default defineConfig({
    testDir: '.',
    testMatch: ['**/*.spec.ts'],
    testIgnore: ['_setup/**', '_fixtures/**'],
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
            command: `${BIN} web --no-open --port 9877`,
            cwd: SPWN_PROJECT,
            port: 9877,
            timeout: 15_000,
            reuseExistingServer: !process.env.CI,
            env: testEnv,
        },
        {
            command: 'npx next dev -p 1420',
            cwd: resolve(REPO_ROOT, 'apps/web'),
            port: 1420,
            timeout: 30_000,
            reuseExistingServer: !process.env.CI,
            env: {
                ...testEnv,
                NEXT_PUBLIC_API_URL: 'http://localhost:9877',
            },
        },
    ],

    globalSetup: './_setup/global-setup.ts',
    globalTeardown: './_setup/global-teardown.ts',

    projects: [
        {
            name: 'chromium',
            use: { ...devices['Desktop Chrome'] },
        },
    ],
});
