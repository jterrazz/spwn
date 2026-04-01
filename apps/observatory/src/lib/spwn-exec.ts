/**
 * Execute spwn CLI commands from API routes.
 * Tries `spwn` from PATH first, falls back to ~/.local/bin/spwn.
 */

import { execSync } from "child_process";
import path from "node:path";
import os from "node:os";

const SPWN_BIN =
  process.env.SPWN_BIN || path.join(os.homedir(), ".local", "bin", "spwn");

function getSpwnPath(): string {
  // Try PATH first
  try {
    execSync("which spwn", { timeout: 3000, stdio: "pipe" });
    return "spwn";
  } catch {
    return SPWN_BIN;
  }
}

let cachedBin: string | null = null;

function bin(): string {
  if (!cachedBin) cachedBin = getSpwnPath();
  return cachedBin;
}

export interface SpwnResult {
  ok: boolean;
  stdout?: string;
  error?: string;
}

/**
 * Run a spwn command and return the result.
 * @param args - Arguments to pass to spwn (e.g. ["down", "w-titan-84721"])
 * @param timeout - Timeout in ms (default 30s)
 */
export function spwnExec(
  args: string[],
  timeout = 30000
): SpwnResult {
  const cmd = `${bin()} ${args.join(" ")}`;
  try {
    const stdout = execSync(cmd, {
      timeout,
      encoding: "utf-8",
      stdio: ["pipe", "pipe", "pipe"],
    });
    return { ok: true, stdout: stdout.trim() };
  } catch (e: unknown) {
    const err = e as { message?: string; stderr?: string };
    return {
      ok: false,
      error: err.stderr || err.message || "Unknown error",
    };
  }
}
