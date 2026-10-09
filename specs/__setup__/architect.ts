import { spawnSync } from 'node:child_process';
import { mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

export const ARCHITECT_IMAGE_OVERRIDE = 'alpine:latest';

// invariant: the product names its architect container spwn-architect whatever the home, so every spec driving it holds
// this lock while its chain runs — the files run in parallel workers, and unlocked they removed each other's container
const LOCK = join(tmpdir(), 'spwn-specs-architect.lock');
const OWNER = join(LOCK, 'pid');

function holderAlive(): boolean {
    const pid = Number(readFileSync(OWNER, { encoding: 'utf8', flag: 'r' }));
    try {
        process.kill(pid, 0);
        return true;
    } catch {
        return false;
    }
}

function tryLock(): boolean {
    try {
        mkdirSync(LOCK);
    } catch {
        let alive = true;
        try {
            alive = holderAlive();
        } catch {
            // value: a lock folder whose pid file is not written yet belongs to a live worker taking it
        }
        if (alive) {
            return false;
        }
        rmSync(LOCK, { force: true, recursive: true });
        return tryLock();
    }
    writeFileSync(OWNER, String(process.pid));
    return true;
}

function removeContainer(): void {
    spawnSync('docker', ['rm', '-f', 'spwn-architect'], { encoding: 'utf8', timeout: 10_000 });
}

function pullOverrideImage(): void {
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
}

/** Runs `chain` with the architect to itself, from none: the override image present, no container, the lock held until the chain ends, its container removed then. */
export async function withoutArchitect<Result>(chain: () => Promise<Result>): Promise<Result> {
    while (!tryLock()) {
        await delay(250);
    }
    try {
        pullOverrideImage();
        removeContainer();
        return await chain();
    } finally {
        removeContainer();
        rmSync(LOCK, { force: true, recursive: true });
    }
}
