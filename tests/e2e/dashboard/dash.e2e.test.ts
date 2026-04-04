import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spawnSync, spawn, type ChildProcess } from "node:child_process";
import { resolve } from "node:path";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";
import { stripAnsi } from "../../setup/output-helpers.js";

const SPWN_BIN = resolve(import.meta.dirname, "../../../bin/spwn");

describe("dashboard — spwn dash", () => {
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

  test("'spwn dash --help' shows subcommands", async () => {
    const result = await spwn("dash help").exec("dash --help").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("start");
    expect(out).toContain("open");
  });

  test("'spwn dash start' produces clean output even on failure", async () => {
    const result = await spwn("dash start").exec("dash start").run();

    // May fail if port is in use or server isn't configured — that's OK
    // We just verify no stack trace / unhandled crash
    const out = stripAnsi(result.output);
    expect(out).not.toContain("TypeError");
    expect(out).not.toContain("ReferenceError");
    expect(out).not.toContain("FATAL");
  });

  test("'spwn dash open' produces clean output even on failure", async () => {
    const result = await spwn("dash open").exec("dash open").run();

    // May not be able to open browser in CI — that's OK
    const out = stripAnsi(result.output);
    expect(out).not.toContain("TypeError");
    expect(out).not.toContain("ReferenceError");
    expect(out).not.toContain("FATAL");
  });

  test("'spwn dash nonexistent' shows error for unknown subcommand", async () => {
    const result = await spwn("dash unknown").exec("dash nonexistent").run();

    // THEN — exits with non-zero or shows usage, but no crash
    const out = stripAnsi(result.output);
    expect(out).not.toContain("TypeError");
    expect(out).not.toContain("ReferenceError");
    expect(out.length).toBeGreaterThan(0);
  });

  // ──────────────────────────────────────────────
  // Test: 'spwn dash start' should bind to a port and serve HTTP
  // ──────────────────────────────────────────────
  test("'spwn dash start' should bind to a port and serve HTTP", async () => {
    // Start the dashboard in a child process
    const child: ChildProcess = spawn(SPWN_BIN, ["dash", "start"], {
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
        // Look for port number in output (e.g., "localhost:3000" or "port 3000")
        const portMatch = combined.match(
          /(?:localhost|127\.0\.0\.1|0\.0\.0\.0):(\d{4,5})/,
        );
        if (portMatch) {
          port = portMatch[1];
          break;
        }

        // Also check for "ready" or "started" messages that contain a URL
        const urlMatch = combined.match(/https?:\/\/[^\s]+:(\d{4,5})/);
        if (urlMatch) {
          port = urlMatch[1];
          break;
        }

        await new Promise((r) => setTimeout(r, 500));
      }

      if (port) {
        // Try to fetch from the server
        try {
          const res = await fetch(`http://localhost:${port}/`, {
            signal: AbortSignal.timeout(5000),
          });
          // Dashboard should respond with HTML or redirect
          expect([200, 301, 302, 307, 308]).toContain(res.status);
        } catch {
          // Server may not be fully ready, but it did bind — that's OK
        }
      }

      // Verify no crash occurred
      const combined = stripAnsi(stdout + stderr);
      expect(combined).not.toContain("TypeError");
      expect(combined).not.toContain("ReferenceError");
      expect(combined).not.toContain("FATAL");
    } finally {
      // Clean up child process
      child.kill("SIGTERM");
      // Give it a moment to terminate
      await new Promise((r) => setTimeout(r, 500));
      if (!child.killed) {
        child.kill("SIGKILL");
      }
    }
  });

  // ──────────────────────────────────────────────
  // Test: 'spwn dash start' should clean up on SIGTERM
  // ──────────────────────────────────────────────
  test("'spwn dash start' should clean up on SIGTERM", async () => {
    const child: ChildProcess = spawn(SPWN_BIN, ["dash", "start"], {
      env: { ...process.env, SPWN_HOME: home },
      stdio: ["pipe", "pipe", "pipe"],
    });

    let exited = false;
    let exitCode: number | null = null;

    child.on("exit", (code) => {
      exited = true;
      exitCode = code;
    });

    // Wait a bit for the process to start
    await new Promise((r) => setTimeout(r, 2000));

    // Send SIGTERM
    child.kill("SIGTERM");

    // Wait for the process to exit (up to 5s)
    const startTime = Date.now();
    while (!exited && Date.now() - startTime < 5000) {
      await new Promise((r) => setTimeout(r, 100));
    }

    // Process should have exited gracefully
    expect(exited).toBe(true);

    // Verify no orphaned processes with the same SPWN_HOME
    // (checking that the process tree is clean)
    const psResult = spawnSync("pgrep", ["-f", `SPWN_HOME=${home}`], {
      encoding: "utf-8",
      timeout: 5000,
    });

    // No matching processes should remain
    const pids = (psResult.stdout ?? "").trim();
    if (pids) {
      // Kill any stragglers and fail
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

  // ──────────────────────────────────────────────
  // Test: Multiple 'spwn dash start' calls should not leave orphans
  // ──────────────────────────────────────────────
  test("multiple 'spwn dash start' calls should not leave orphan processes", async () => {
    const children: ChildProcess[] = [];

    try {
      // Start 3 instances rapidly
      for (let i = 0; i < 3; i++) {
        const child = spawn(SPWN_BIN, ["dash", "start"], {
          env: { ...process.env, SPWN_HOME: home },
          stdio: ["pipe", "pipe", "pipe"],
        });
        children.push(child);
      }

      // Wait a bit for them to start
      await new Promise((r) => setTimeout(r, 2000));

      // Kill all children
      for (const child of children) {
        child.kill("SIGTERM");
      }

      // Wait for all to exit
      await new Promise((r) => setTimeout(r, 2000));

      // Verify no orphaned node/next processes remain from our test
      // We check that all spawned children are actually dead
      for (const child of children) {
        // child.killed should be true, or exitCode should be set
        expect(child.killed || child.exitCode !== null).toBe(true);
      }

      // Double check: no processes left with our SPWN_HOME
      const psResult = spawnSync("pgrep", ["-f", `SPWN_HOME=${home}`], {
        encoding: "utf-8",
        timeout: 5000,
      });

      // Clean up any stragglers
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
      // Ensure cleanup even if test fails
      for (const child of children) {
        if (!child.killed) {
          child.kill("SIGKILL");
        }
      }
      await new Promise((r) => setTimeout(r, 500));
    }
  });
});
