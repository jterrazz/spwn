import { execSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

export default async function globalTeardown() {
    console.log('\n[global-teardown] Cleaning up...');

    const configPath = process.env.SPWN_TEST_CONFIG;
    let config: { spwnHome?: string; spwnProject?: string; testLabel?: string } = {};
    if (configPath) {
        try {
            config = JSON.parse(readFileSync(configPath, 'utf8'));
        } catch {
            /* Best effort */
        }
    }

    // Destroy only Docker containers created by this Playwright run.
    if (config.testLabel) {
        try {
            execSync(
                `docker ps --filter "label=sh.spwn.test.run=${config.testLabel}" -q | xargs -r docker rm -f`,
                {
                    stdio: 'ignore',
                    timeout: 10_000,
                },
            );
        } catch {
            /* Best effort */
        }
    }

    if (config.spwnHome) {
        try {
            execSync(`rm -rf "${config.spwnHome}"`, { stdio: 'ignore', timeout: 5000 });
        } catch {
            /* Best effort */
        }
    }

    if (config.spwnProject) {
        try {
            execSync(`rm -rf "${config.spwnProject}"`, { stdio: 'ignore', timeout: 5000 });
        } catch {
            /* Best effort */
        }
    }

    // Playwright kills the webServer processes automatically
    console.log('[global-teardown] Done ✓\n');
}
