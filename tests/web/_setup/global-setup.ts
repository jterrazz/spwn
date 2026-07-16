import { execSync } from 'node:child_process';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const currentDir = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(currentDir, '../../..');

/**
 * Build the binary so the Tauri sidecar exists. Examples are
 * installed lazily through /api/examples by individual tests when
 * they need them. Servers are managed by playwright.config.ts.
 */
export default async function globalSetup() {
    console.log('\n[global-setup] Building spwn binary...');
    execSync('make build', { cwd: REPO_ROOT, stdio: 'inherit' });
    console.log('[global-setup] Ready ✓\n');
}
