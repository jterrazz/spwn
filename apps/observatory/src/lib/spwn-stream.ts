/**
 * Streaming version of spwnExec — uses spawn instead of execSync
 * so stdout can be piped as a ReadableStream response.
 */

import { spawn } from "child_process";
import { execSync } from "child_process";
import path from "node:path";
import os from "node:os";

const SPWN_BIN =
  process.env.SPWN_BIN || path.join(os.homedir(), ".local", "bin", "spwn");

function getSpwnPath(): string {
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

/**
 * Spawn a spwn command and return a ReadableStream of its stdout.
 * The stream sends text chunks as they arrive from the process.
 */
export function spwnStream(
  args: string[],
  timeout = 120000
): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();

  return new ReadableStream({
    start(controller) {
      const proc = spawn(bin(), args, {
        stdio: ["pipe", "pipe", "pipe"],
        env: { ...process.env },
      });

      let stderr = "";

      const timer = setTimeout(() => {
        proc.kill("SIGTERM");
        controller.enqueue(encoder.encode("\n[timeout]"));
        controller.close();
      }, timeout);

      proc.stdout.on("data", (chunk: Buffer) => {
        controller.enqueue(new Uint8Array(chunk));
      });

      proc.stderr.on("data", (chunk: Buffer) => {
        stderr += chunk.toString();
      });

      proc.on("close", (code) => {
        clearTimeout(timer);
        if (code !== 0 && stderr) {
          controller.enqueue(encoder.encode(`\n[error: ${stderr.trim()}]`));
        }
        controller.close();
      });

      proc.on("error", (err) => {
        clearTimeout(timer);
        controller.enqueue(encoder.encode(`\n[error: ${err.message}]`));
        controller.close();
      });
    },
  });
}
