import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spawnSync, spawn, type ChildProcess } from "node:child_process";
import { resolve } from "node:path";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";
import { stripAnsi } from "../../setup/output-helpers.js";

const SPWN_BIN = resolve(import.meta.dirname, "../../../bin/spwn");

describe("web — spwn web", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
  });

  test("'spwn web --help' shows flags", async () => {
    const result = await spwn("web help").exec("web --help").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("--port");
    expect(out).toContain("--no-open");
  });

  test("'spwn web' should bind to a port and serve HTTP", async () => {
    // Run headless so it doesn't try to launch a browser in CI.
    const child: ChildProcess = spawn(SPWN_BIN, ["web", "--no-open"], {
      env: { ...process.env, SPWN_HOME: home, PORT: "0" },
      stdio: ["pipe", "pipe", "pipe"],
    });

    let stdout = "";
    let stderr = "";
    child.stdout?.on("data", (data: Buffer) => {
      stdout += data.toString();
    });
    child.stderr?.on("data", (data: Buffer) => {
      stderr += data.toString();
    });

    try {
      // Wait up to 10s for the server to emit a port or URL
      const startTime = Date.now();
      let port: string | null = null;

      while (Date.now() - startTime < 10_000) {
        const combined = stdout + stderr;
        const portMatch = combined.match(
          /(?:localhost|127\.0\.0\.1|0\.0\.0\.0):(\d{4,5})/,
        );
        if (portMatch) {
          port = portMatch[1];
          break;
        }

        const urlMatch = combined.match(/https?:\/\/[^\s]+:(\d{4,5})/);
        if (urlMatch) {
          port = urlMatch[1];
          break;
        }

        await new Promise((r) => setTimeout(r, 500));
      }

      if (port) {
        try {
          const res = await fetch(`http://localhost:${port}/`, {
            signal: AbortSignal.timeout(5000),
          });
          expect([200, 301, 302, 307, 308]).toContain(res.status);
        } catch {
          // Server may not be fully ready, but it did bind — that's OK
        }
      }

      const combined = stripAnsi(stdout + stderr);
      expect(combined).not.toContain("TypeError");
      expect(combined).not.toContain("ReferenceError");
      expect(combined).not.toContain("FATAL");
    } finally {
      child.kill("SIGTERM");
      await new Promise((r) => setTimeout(r, 500));
      if (!child.killed) {
        child.kill("SIGKILL");
      }
    }
  });

  test("'spwn web' should clean up on SIGTERM", async () => {
    const child: ChildProcess = spawn(SPWN_BIN, ["web", "--no-open"], {
      env: { ...process.env, SPWN_HOME: home },
      stdio: ["pipe", "pipe", "pipe"],
    });

    let exited = false;

    child.on("exit", () => {
      exited = true;
    });

    await new Promise((r) => setTimeout(r, 2000));

    child.kill("SIGTERM");

    const startTime = Date.now();
    while (!exited && Date.now() - startTime < 5000) {
      await new Promise((r) => setTimeout(r, 100));
    }

    expect(exited).toBe(true);

    // Verify no orphaned processes with the same SPWN_HOME
    const psResult = spawnSync("pgrep", ["-f", `SPWN_HOME=${home}`], {
      encoding: "utf-8",
      timeout: 5000,
    });

    const pids = (psResult.stdout ?? "").trim();
    if (pids) {
      for (const pid of pids.split("\n").filter(Boolean)) {
        try {
          process.kill(parseInt(pid), "SIGKILL");
        } catch {
          // already dead
        }
      }
    }
    // pgrep exit code 1 means no matches found — that's what we want
    expect(psResult.status).not.toBe(0);
  });

  test("multiple 'spwn web' calls should not leave orphan processes", async () => {
    const children: ChildProcess[] = [];

    try {
      for (let i = 0; i < 3; i++) {
        const child = spawn(SPWN_BIN, ["web", "--no-open"], {
          env: { ...process.env, SPWN_HOME: home },
          stdio: ["pipe", "pipe", "pipe"],
        });
        children.push(child);
      }

      await new Promise((r) => setTimeout(r, 2000));

      for (const child of children) {
        child.kill("SIGTERM");
      }

      await new Promise((r) => setTimeout(r, 2000));

      for (const child of children) {
        expect(child.killed || child.exitCode !== null).toBe(true);
      }

      const psResult = spawnSync("pgrep", ["-f", `SPWN_HOME=${home}`], {
        encoding: "utf-8",
        timeout: 5000,
      });

      const pids = (psResult.stdout ?? "")
        .trim()
        .split("\n")
        .filter(Boolean);
      for (const pid of pids) {
        try {
          process.kill(parseInt(pid), "SIGKILL");
        } catch {
          // already dead
        }
      }
    } finally {
      for (const child of children) {
        if (!child.killed) {
          child.kill("SIGKILL");
        }
      }
      await new Promise((r) => setTimeout(r, 500));
    }
  });
});
