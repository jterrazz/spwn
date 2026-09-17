import { execSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

type TestConfig = {
    spwnHome?: string | undefined;
    spwnProject?: string | undefined;
    testLabel?: string | undefined;
};

/** The run's own config, or an empty one when it cannot be read. */
function readConfig(path: string | undefined): TestConfig {
    if (path === undefined || path === '') {
        return {};
    }
    try {
        const parsed: unknown = JSON.parse(readFileSync(path, 'utf8'));
        if (typeof parsed !== 'object' || parsed === null) {
            return {};
        }
        const fields: Record<string, unknown> = { ...parsed };
        const text = (key: string) =>
            typeof fields[key] === 'string' && fields[key] !== '' ? fields[key] : undefined;
        return {
            spwnHome: text('spwnHome'),
            spwnProject: text('spwnProject'),
            testLabel: text('testLabel'),
        };
    } catch {
        return {};
    }
}

/** Best effort: whatever is already gone stays gone. */
function shell(command: string, timeout: number) {
    try {
        execSync(command, { stdio: 'ignore', timeout });
    } catch {
        /* Best effort */
    }
}

export default function globalTeardown() {
    console.log('\n[global-teardown] Cleaning up...');

    const config = readConfig(process.env.SPWN_TEST_CONFIG);

    // Destroy only Docker containers created by this Playwright run.
    if (config.testLabel !== undefined) {
        shell(
            `docker ps --filter "label=sh.spwn.test.run=${config.testLabel}" -q | xargs -r docker rm -f`,
            10_000,
        );
    }
    if (config.spwnHome !== undefined) {
        shell(`rm -rf "${config.spwnHome}"`, 5000);
    }
    if (config.spwnProject !== undefined) {
        shell(`rm -rf "${config.spwnProject}"`, 5000);
    }

    // Playwright kills the webServer processes automatically
    console.log('[global-teardown] Done ✓\n');
}
