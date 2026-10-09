import { spawnSync } from 'node:child_process';

export const ARCHITECT_IMAGE_OVERRIDE = 'alpine:latest';

/** The architect container has one fixed name, so a run starts from none: the override image present, a container a crashed run left behind gone. */
export function withoutArchitect(): void {
    const present = spawnSync('docker', ['image', 'inspect', ARCHITECT_IMAGE_OVERRIDE], {
        encoding: 'utf8',
        timeout: 5000,
    });
    if (present.status !== 0) {
        spawnSync('docker', ['pull', ARCHITECT_IMAGE_OVERRIDE], {
            encoding: 'utf8',
            timeout: 60_000,
        });
    }
    spawnSync('docker', ['rm', '-f', 'spwn-architect'], { encoding: 'utf8', timeout: 10_000 });
}
