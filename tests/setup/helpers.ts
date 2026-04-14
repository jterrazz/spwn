import { mkdtempSync, mkdirSync, writeFileSync, existsSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

/**
 * Create an isolated SPWN_HOME directory with the required subdirectories.
 * Each call creates a unique temporary directory under the system temp path.
 *
 * @returns Absolute path to the new SPWN_HOME directory
 */
export function createSpwnHome(): string {
  const dir = mkdtempSync(join(tmpdir(), "spwn-test-"));
  mkdirSync(join(dir, "worlds"), { recursive: true });
  mkdirSync(join(dir, "agents"), { recursive: true });
  return dir;
}

/**
 * Create a minimal agent Mind with the standard 6-layer directory structure.
 * Writes a default identity file so the agent is immediately usable.
 *
 * @param spwnHome - Path to the SPWN_HOME directory
 * @param name - Agent name (used as directory name under agents/)
 */
export function createAgent(spwnHome: string, name: string): void {
  const agentDir = join(spwnHome, "agents", name);
  // Current Mind layout: core, skills, knowledge, playbooks, journal
  // (matches foundation.MindLayers)
  const layers = ["core", "skills", "knowledge", "playbooks", "journal"];
  for (const layer of layers) {
    mkdirSync(join(agentDir, layer), { recursive: true });
  }
  writeFileSync(
    join(agentDir, "core", "profile.md"),
    `# ${name}\n\nYou are a test agent named ${name}.\n\n## Purpose\n\nTest automation.\n\n## Traits\n\n- Reliable\n- Systematic\n`,
  );
  writeFileSync(join(agentDir, "agent.yaml"), `role: worker\n`);
}

/**
 * Assert the spwn binary exists and is executable.
 * Throws a descriptive error if not found.
 */
export function assertBinaryExists(): void {
  const binPath = resolve(import.meta.dirname, "../../bin/spwn");
  if (!existsSync(binPath)) {
    throw new Error(
      `spwn binary not found at ${binPath}.\n` +
        `Make sure to build the project first: npm run build\n` +
        `Or check that the path is correct relative to tests/setup/helpers.ts`,
    );
  }
}

/**
 * Retry a function up to maxRetries times, with a delay between attempts.
 * Useful for Docker operations that may be flaky.
 */
export function retry<T>(
  fn: () => T,
  maxRetries = 3,
  delayMs = 1000,
): T {
  let lastError: unknown;
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      return fn();
    } catch (err) {
      lastError = err;
      if (attempt < maxRetries) {
        sleepMs(delayMs);
      }
    }
  }
  throw lastError;
}

/**
 * Run async tasks with bounded concurrency.
 * Executes up to `maxConcurrency` tasks at a time, waiting for a slot
 * to open before launching the next one.
 *
 * @param tasks - Array of async task factories
 * @param maxConcurrency - Maximum number of concurrent tasks
 */
export async function runConcurrently(
  tasks: (() => Promise<void>)[],
  maxConcurrency: number,
): Promise<void> {
  const executing = new Set<Promise<void>>();

  for (const task of tasks) {
    const p = task().then(() => {
      executing.delete(p);
    });
    executing.add(p);

    if (executing.size >= maxConcurrency) {
      await Promise.race(executing);
    }
  }

  await Promise.all(executing);
}

/** Synchronous sleep helper */
function sleepMs(ms: number): void {
  spawnSync("sleep", [String(ms / 1000)], { timeout: ms + 1000 });
}
