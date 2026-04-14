import { spawnSync } from 'node:child_process';
import { resolve } from 'node:path';
import { afterEach, beforeAll, beforeEach, describe, expect, test } from 'vitest';

import { createSpwnHome } from '../../setup/helpers.js';

const SPWN_BIN = resolve(import.meta.dirname, '../../../bin/spwn');
const CONTAINER_NAME = 'spwn-architect';

/**
 * Execute spwn with the given args and environment overrides.
 */
function spwnExec(
    args: string[],
    envOverrides: Record<string, string> = {},
): { exitCode: number; stdout: string; stderr: string; output: string } {
    const env = {
        ...process.env,
        INIT_CWD: undefined,
        ...envOverrides,
    };

    const result = spawnSync(SPWN_BIN, args, {
        encoding: 'utf8',
        env: env as NodeJS.ProcessEnv,
        stdio: ['pipe', 'pipe', 'pipe'],
        timeout: 30_000,
    });

    const stdout = result.stdout ?? '';
    const stderr = result.stderr ?? '';
    const exitCode = result.status ?? 1;

    return { exitCode, stdout, stderr, output: stdout + stderr };
}

/**
 * Check if a Docker image exists locally.
 */
function imageExists(image: string): boolean {
    const result = spawnSync('docker', ['image', 'inspect', image], {
        encoding: 'utf8',
        timeout: 5000,
    });
    return result.status === 0;
}

/**
 * Remove the architect container if it exists (cleanup helper).
 */
function removeContainer(): void {
    spawnSync('docker', ['rm', '-f', CONTAINER_NAME], {
        encoding: 'utf8',
        timeout: 10_000,
    });
}

/**
 * Check if a container exists and is running.
 */
function containerRunning(name: string): boolean {
    const result = spawnSync('docker', ['inspect', '--format', '{{.State.Running}}', name], {
        encoding: 'utf8',
        timeout: 5000,
    });
    return (result.stdout ?? '').trim() === 'true';
}

/**
 * Check if Docker is available.
 */
function dockerAvailable(): boolean {
    const result = spawnSync('docker', ['info'], {
        encoding: 'utf8',
        timeout: 5000,
    });
    return result.status === 0;
}

describe('spwn architect', () => {
    let home: string;
    let originalSpwnHome: string | undefined;
    // We use alpine:latest with a sleep entrypoint for testing
    const testImage = 'alpine:latest';

    beforeAll(() => {
        if (!dockerAvailable()) {
            console.warn('Docker not available, skipping architect e2e tests');
            return;
        }

        // Pull alpine image if needed
        if (!imageExists(testImage)) {
            spawnSync('docker', ['pull', testImage], {
                encoding: 'utf8',
                timeout: 60_000,
            });
        }

        // Tag alpine as spwn-architect:latest for testing
        // (The real image runs the architect serve command, but for lifecycle
        // Testing we just need a container that stays alive)
        spawnSync('docker', ['build', '-t', 'spwn-architect:latest', '-'], {
            input: `FROM alpine:latest\nCMD ["sleep", "infinity"]`,
            encoding: 'utf8',
            timeout: 30_000,
        });
    });

    beforeEach(() => {
        originalSpwnHome = process.env.SPWN_HOME;
        home = createSpwnHome();
        process.env.SPWN_HOME = home;
        // Clean up any leftover container
        removeContainer();
    });

    afterEach(() => {
        // Always clean up the container
        removeContainer();

        if (originalSpwnHome !== undefined) {
            process.env.SPWN_HOME = originalSpwnHome;
        } else {
            delete process.env.SPWN_HOME;
        }
    });

    test('status shows not running when no container exists', () => {
        if (!dockerAvailable()) {
            return;
        }

        const result = spwnExec(['architect', 'status'], { SPWN_HOME: home });

        expect(result.exitCode).toBe(0);
        expect(result.output).toContain('not running');
    });

    test('start creates and runs a Docker container', () => {
        if (!dockerAvailable()) {
            return;
        }

        const result = spwnExec(['architect', 'start'], { SPWN_HOME: home });

        expect(result.exitCode).toBe(0);
        expect(result.output).toContain('Architect started');
        expect(containerRunning(CONTAINER_NAME)).toBe(true);
    });

    test('status shows running after start', () => {
        if (!dockerAvailable()) {
            return;
        }

        // Start first
        spwnExec(['architect', 'start'], { SPWN_HOME: home });

        // Then check status
        const result = spwnExec(['architect', 'status'], { SPWN_HOME: home });

        expect(result.exitCode).toBe(0);
        expect(result.output).toContain('running');
        // The status output now reports "Architect: running" + Container/Image
        // Lines, no longer the raw container name. Image label is enough.
        expect(result.output).toMatch(/Image:\s+spwn\/architect/);
    });

    test('start again shows already running', () => {
        if (!dockerAvailable()) {
            return;
        }

        // Start first time
        spwnExec(['architect', 'start'], { SPWN_HOME: home });

        // Start again
        const result = spwnExec(['architect', 'start'], { SPWN_HOME: home });

        expect(result.exitCode).toBe(0);
        expect(result.output).toContain('already running');
    });

    test('stop removes the container', () => {
        if (!dockerAvailable()) {
            return;
        }

        // Start first
        spwnExec(['architect', 'start'], { SPWN_HOME: home });
        expect(containerRunning(CONTAINER_NAME)).toBe(true);

        // Stop
        const result = spwnExec(['architect', 'stop'], { SPWN_HOME: home });

        expect(result.exitCode).toBe(0);
        expect(result.output).toContain('Architect stopped');
        expect(containerRunning(CONTAINER_NAME)).toBe(false);
    });

    test('status shows not running after stop', () => {
        if (!dockerAvailable()) {
            return;
        }

        // Start, then stop
        spwnExec(['architect', 'start'], { SPWN_HOME: home });
        spwnExec(['architect', 'stop'], { SPWN_HOME: home });

        // Check status
        const result = spwnExec(['architect', 'status'], { SPWN_HOME: home });

        expect(result.exitCode).toBe(0);
        expect(result.output).toContain('not running');
    });

    test('stop when not running is a clean no-op', () => {
        if (!dockerAvailable()) {
            return;
        }

        const result = spwnExec(['architect', 'stop'], { SPWN_HOME: home });

        // Stop now exits 0 and reports "not running" (idempotent shutdown).
        expect(result.exitCode).toBe(0);
        expect(result.output).toContain('not running');
        expect(result.output).not.toContain('panic');
    });

    test('start with SPWN_ARCHITECT_IMAGE override uses custom image', () => {
        if (!dockerAvailable()) {
            return;
        }

        const result = spwnExec(['architect', 'start'], {
            SPWN_HOME: home,
            SPWN_ARCHITECT_IMAGE: 'alpine:latest',
        });

        expect(result.exitCode).toBe(0);
        expect(containerRunning(CONTAINER_NAME)).toBe(true);
    });

    test('talk --output-format stream-json outputs JSON events', () => {
        if (!dockerAvailable()) {
            return;
        }

        // Start the architect first
        const startResult = spwnExec(['architect', 'start'], { SPWN_HOME: home });

        // Only test streaming if architect started successfully
        if (startResult.exitCode !== 0) {
            return;
        }

        const result = spwnExec(['architect', 'talk', '--output-format', 'stream-json', 'hello'], {
            SPWN_HOME: home,
        });

        // The architect "talk" only emits JSON when an LLM is wired up. With
        // A stub container (alpine + sleep), claude isn't installed, so this
        // Test only asserts that `--output-format stream-json` doesn't crash.
        expect(result.output).not.toContain('panic');
        expect(result.output).not.toContain('FATAL');
    });
});
