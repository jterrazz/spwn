import { execSync } from 'node:child_process';
import { resolve } from 'node:path';

const REPO_ROOT = resolve(import.meta.dirname, '../../..');

/**
 * Build the binary so the Tauri sidecar exists. Examples are
 * installed lazily through /api/examples by individual tests when
 * they need them. Servers are managed by playwright.config.ts.
 */
export default function globalSetup() {
    console.log('\n[global-setup] Building spwn binary...');
    execSync('make build', { cwd: REPO_ROOT, stdio: 'inherit' });
    console.log('[global-setup] Ready ✓\n');
}
