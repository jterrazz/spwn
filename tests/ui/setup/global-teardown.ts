import { execSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

export default async function globalTeardown() {
    console.log('\n[global-teardown] Cleaning up...');

    // Destroy any Docker containers created during tests
    try {
        execSync('docker ps --filter "label=spwn.kind=world" -q | xargs -r docker rm -f', {
            stdio: 'ignore',
            timeout: 10_000,
        });
    } catch {
        /* Best effort */
    }

    // Clean up the isolated SPWN_HOME temp directory
    const configPath = process.env.SPWN_TEST_CONFIG;
    if (configPath) {
        try {
            const config = JSON.parse(readFileSync(configPath, 'utf8'));
            if (config.spwnHome) {
                execSync(`rm -rf "${config.spwnHome}"`, { stdio: 'ignore', timeout: 5000 });
            }
        } catch {
            /* Best effort */
        }
    }

    // Playwright kills the webServer processes automatically
    console.log('[global-teardown] Done ✓\n');
}
