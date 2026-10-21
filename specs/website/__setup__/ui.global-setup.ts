import { execFileSync, spawn } from 'node:child_process';
import { createServer } from 'node:net';
import { join } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';
import type { TestProject } from 'vitest/node';

declare module 'vitest' {
    export interface ProvidedContext {
        spwnApiPort: number;
        spwnUiUrl: string;
    }
}

const WEB = join(import.meta.dirname, '../../../apps/web');
const READY_BUDGET_MS = 60_000;
const READY_POLL_MS = 200;

async function freePort(): Promise<number> {
    const server = createServer();
    await new Promise<void>((resolve) => {
        server.listen(0, '127.0.0.1', resolve);
    });
    const address = server.address();
    await new Promise((resolve) => {
        server.close(resolve);
    });
    if (address === null || typeof address === 'string') {
        throw new Error('the loopback interface handed no port');
    }
    return address.port;
}

async function answers(url: string): Promise<boolean> {
    try {
        const response = await fetch(url);
        return response.ok;
    } catch {
        return false;
    }
}

// invariant: the UI is built and served once per run, as `spwn web` serves it, with the one API address every file's API then listens on: the client reads that address from the build, so the files take turns on it.
export default async function setup(project: TestProject): Promise<() => void> {
    const apiPort = await freePort();
    execFileSync('pnpm', ['run', 'build'], {
        cwd: WEB,
        env: { ...process.env, NEXT_PUBLIC_API_URL: `http://127.0.0.1:${apiPort}` },
    });
    const uiPort = await freePort();
    const ui = spawn(
        'pnpm',
        ['run', 'start', '--hostname', '127.0.0.1', '--port', String(uiPort)],
        {
            cwd: WEB,
            detached: true,
            stdio: 'ignore',
        },
    );
    const stop = (): void => {
        if (ui.pid !== undefined && ui.exitCode === null) {
            process.kill(-ui.pid, 'SIGTERM');
        }
    };
    const url = `http://127.0.0.1:${uiPort}`;
    const deadline = Date.now() + READY_BUDGET_MS;
    while (!(await answers(url))) {
        if (ui.exitCode !== null || Date.now() > deadline) {
            stop();
            throw new Error(
                `next start in ${WEB} did not answer on ${url} within ${READY_BUDGET_MS} ms`,
            );
        }
        await delay(READY_POLL_MS);
    }
    project.provide('spwnApiPort', apiPort);
    project.provide('spwnUiUrl', url);
    return stop;
}
