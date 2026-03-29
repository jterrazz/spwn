import { spawnSync } from "node:child_process";
import { resolve } from "node:path";

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
