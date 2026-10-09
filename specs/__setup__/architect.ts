import { spawnSync } from 'node:child_process';
import { mkdirSync, readFileSync, rmSync, statSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';

export const ARCHITECT_IMAGE_OVERRIDE = 'alpine:latest';

// invariant: the product names its architect container spwn-architect whatever the home, so every spec driving it holds
// this lock while its chain runs — the files run in parallel workers, and unlocked they removed each other's container
const LOCK = join(tmpdir(), 'spwn-specs-architect.lock');
const OWNER = join(LOCK, 'pid');
const PID_WRITE_GRACE_MS = 5000;

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
        let alive: boolean;
        try {
            alive = holderAlive();
        } catch {
            // invariant: a worker writes its pid right after taking the folder, so a folder without one past the grace was left by a worker that died between the two; a folder already gone is taken on the next turn
            const taken = statSync(LOCK, { throwIfNoEntry: false });
            alive = taken === undefined || Date.now() - taken.mtimeMs < PID_WRITE_GRACE_MS;
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
