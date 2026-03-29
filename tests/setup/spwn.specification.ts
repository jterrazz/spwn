import { spawnSync } from "node:child_process";
import { resolve } from "node:path";
import { createSpwnHome, createAgent } from "./helpers.js";
import { UniverseAssertion } from "./universe-assertion.js";
import { MindAssertion } from "./mind-assertion.js";
import { StateAssertion } from "./state-assertion.js";

// Build the binary path
const SPWN_BIN = resolve(import.meta.dirname, "../../bin/spwn");

/**
 * Simple specification runner for the spwn CLI binary.
 *
 * The spwn binary writes user-facing output to stderr (unix convention:
 * stdout for data, stderr for status). The @jterrazz/test ExecAdapter
 * discards stderr on success, so we use a custom runner that captures
 * both streams. The `output` field merges stdout + stderr for assertions.
 */
export function spwn(label: string) {
  let args = "";

  return {
    exec(cmdArgs: string | string[]) {
      args = Array.isArray(cmdArgs) ? cmdArgs.join(" ") : cmdArgs;
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
        encoding: "utf-8",
        env: env as NodeJS.ProcessEnv,
        stdio: ["pipe", "pipe", "pipe"],
        timeout: 30_000,
      });

      const stdout = result.stdout ?? "";
      const stderr = result.stderr ?? "";
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
    encoding: "utf-8",
    env: env as NodeJS.ProcessEnv,
    stdio: ["pipe", "pipe", "pipe"],
    timeout,
  });

  const stdout = result.stdout ?? "";
  const stderr = result.stderr ?? "";
  const exitCode = result.status ?? 1;

  return {
    exitCode,
    stdout,
    stderr,
    output: stdout + stderr,
  };
}

/**
 * Parse a universe ID (u-{name}-{5digits}) from command output.
 */
export function parseUniverseId(output: string): string | null {
  const match = output.match(/u-[\w]+-\d{5}/);
  return match ? match[0] : null;
}

/**
 * Parse all universe IDs from output (e.g., from list command).
 */
export function parseAllUniverseIds(output: string): string[] {
  const matches = output.matchAll(/u-[\w]+-\d{5}/g);
  return [...matches].map((m) => m[0]);
}

export interface TestContext {
  home: string;
  spwn: (args: string[], timeout?: number) => {
    exitCode: number;
    stdout: string;
    stderr: string;
    output: string;
  };
  /** Inspect a running universe container */
  universe: (universeId: string) => UniverseAssertion;
  /** Inspect an agent Mind on disk */
  mind: (agentName: string) => MindAssertion;
  /** Inspect the state.json file */
  state: () => StateAssertion;
  /** Destroy all active universes and clean up temp directory */
  cleanup: () => void;
}

/**
 * Create an isolated test context with its own SPWN_HOME.
 * Each context uses SPWN_BASE_IMAGE=spwn-test:latest.
 */
export function createTestContext(): TestContext {
  const home = createSpwnHome();
  createAgent(home, "neo");

  const envOverrides = {
    SPWN_HOME: home,
    SPWN_BASE_IMAGE: "spwn-test:latest",
  };

  const ctx: TestContext = {
    home,
    spwn: (args: string[], timeout = 30_000) =>
      spwnWithEnv(args, envOverrides, timeout),
    universe: (universeId: string) =>
      new UniverseAssertion(universeId, home),
    mind: (agentName: string) => new MindAssertion(home, agentName),
    state: () => new StateAssertion(home),
    cleanup: () => {
      // Destroy all active universes
      const listResult = spwnWithEnv(["universe", "list"], envOverrides);
      const ids = parseAllUniverseIds(listResult.output);
      for (const id of ids) {
        spwnWithEnv(["universe", "destroy", id], envOverrides, 30_000);
      }
      // Clean up temp dir
      try {
        spawnSync("rm", ["-rf", home], { timeout: 5000 });
      } catch {
        // best effort
      }
    },
  };

  return ctx;
}
