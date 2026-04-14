import { spawnSync } from 'node:child_process';
import { resolve } from 'node:path';

import { createAgent, createSpwnHome } from './helpers.js';
import { MindAssertion } from './mind-assertion.js';
import { StateAssertion } from './state-assertion.js';
import { WorldAssertion } from './world-assertion.js';

// Build the binary path
const SPWN_BIN = resolve(import.meta.dirname, '../../bin/spwn');

/**
 * Execute the spwn binary with custom environment variables.
 */
function spwnWithEnv(
    args: string[],
    envOverrides: Record<string, string>,
    timeout = 30_000,
): {
    exitCode: number;
    stdout: string;
    stderr: string;
    output: string;
} {
    const env = {
        ...process.env,
        INIT_CWD: undefined,
        ...envOverrides,
    };

    const result = spawnSync(SPWN_BIN, args, {
        encoding: 'utf8',
        env: env as NodeJS.ProcessEnv,
        stdio: ['pipe', 'pipe', 'pipe'],
        timeout,
    });

    const stdout = result.stdout ?? '';
    const stderr = result.stderr ?? '';
    const exitCode = result.status ?? 1;

    return {
        exitCode,
        stdout,
        stderr,
        output: stdout + stderr,
    };
}

/**
 * Parse a world ID from command output.
 *
 * New format: spwn-world-{name}-{5digits}
 * Legacy:     w-{name}-{5digits} (kept for state files written before v1.1.0)
 */
const WORLD_ID_RE = /(?:spwn-world|w)-\w+-\d{5}/;
const WORLD_ID_RE_GLOBAL = /(?:spwn-world|w)-\w+-\d{5}/g;

function parseAllWorldIds(output: string): string[] {
    const matches = output.matchAll(WORLD_ID_RE_GLOBAL);
    return [...matches].map((m) => m[0]);
}

export interface TestContext {
    home: string;
    spwn: (
        args: string[],
        timeout?: number,
    ) => {
        exitCode: number;
        stdout: string;
        stderr: string;
        output: string;
    };
    /** Inspect a running world container */
    world: (worldId: string) => WorldAssertion;
    /** Inspect an agent Mind on disk */
    mind: (agentName: string) => MindAssertion;
    /** Inspect the state.json file */
    state: () => StateAssertion;
    /** Destroy all active worlds and clean up temp directory */
    cleanup: () => void;
}

/**
 * Simple specification runner for the spwn CLI binary.
 *
 * The spwn binary writes user-facing output to stderr (unix convention:
 * stdout for data, stderr for status). The @jterrazz/test ExecAdapter
 * discards stderr on success, so we use a custom runner that captures
 * both streams. The `output` field merges stdout + stderr for assertions.
 */
export function spwn(_label: string) {
    let args = '';

    return {
        exec(cmdArgs: string | string[]) {
            args = Array.isArray(cmdArgs) ? cmdArgs.join(' ') : cmdArgs;
            return this;
        },

        async run(): Promise<{
            exitCode: number;
            stdout: string;
            stderr: string;
            output: string;
        }> {
            const env = {
                ...process.env,
                INIT_CWD: undefined,
            };

            const result = spawnSync(SPWN_BIN, args.split(/\s+/).filter(Boolean), {
                encoding: 'utf8',
                env: env as NodeJS.ProcessEnv,
                stdio: ['pipe', 'pipe', 'pipe'],
                timeout: 30_000,
            });

            const stdout = result.stdout ?? '';
            const stderr = result.stderr ?? '';
            const exitCode = result.status ?? 1;

            return {
                exitCode,
                stdout,
                stderr,
                output: stdout + stderr,
            };
        },
    };
}

export function parseWorldId(output: string): null | string {
    const match = output.match(WORLD_ID_RE);
    return match ? match[0] : null;
}

/**
 * Create an isolated test context with its own SPWN_HOME.
 * Each context uses SPWN_BASE_IMAGE=spwn-test:latest.
 */
export function createTestContext(): TestContext {
    const home = createSpwnHome();
    createAgent(home, 'neo');

    const envOverrides = {
        SPWN_HOME: home,
        SPWN_BASE_IMAGE: 'spwn-test:latest',
    };

    const ctx: TestContext = {
        home,
        spwn: (args: string[], timeout = 30_000) => spwnWithEnv(args, envOverrides, timeout),
        world: (worldId: string) => new WorldAssertion(worldId, home),
        mind: (agentName: string) => new MindAssertion(home, agentName),
        state: () => new StateAssertion(),
        cleanup: () => {
            // Destroy all active worlds
            const listResult = spwnWithEnv(['world', 'list'], envOverrides);
            const ids = parseAllWorldIds(listResult.output);
            for (const id of ids) {
                spwnWithEnv(['world', 'destroy', id], envOverrides, 30_000);
            }
            // Clean up temp dir
            try {
                spawnSync('rm', ['-rf', home], { timeout: 5000 });
            } catch {
                // Best effort
            }
        },
    };

    return ctx;
}
