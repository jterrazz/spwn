import { describe, test, expect, beforeEach, afterEach, beforeAll } from "vitest";
import { spawnSync } from "node:child_process";
import { resolve } from "node:path";
import { createSpwnHome } from "../../setup/helpers.js";

const SPWN_BIN = resolve(import.meta.dirname, "../../../bin/spwn");
const CONTAINER_NAME = "spwn-architect";

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
    encoding: "utf-8",
    env: env as NodeJS.ProcessEnv,
    stdio: ["pipe", "pipe", "pipe"],
    timeout: 30_000,
  });

  const stdout = result.stdout ?? "";
  const stderr = result.stderr ?? "";
  const exitCode = result.status ?? 1;

  return { exitCode, stdout, stderr, output: stdout + stderr };
}

/**
 * Check if a Docker image exists locally.
 */
function imageExists(image: string): boolean {
  const result = spawnSync("docker", ["image", "inspect", image], {
    encoding: "utf-8",
    timeout: 5_000,
  });
  return result.status === 0;
}

/**
 * Remove the architect container if it exists (cleanup helper).
 */
function removeContainer(): void {
  spawnSync("docker", ["rm", "-f", CONTAINER_NAME], {
    encoding: "utf-8",
    timeout: 10_000,
  });
}

/**
 * Check if a container exists and is running.
 */
function containerRunning(name: string): boolean {
  const result = spawnSync(
    "docker",
    ["inspect", "--format", "{{.State.Running}}", name],
    { encoding: "utf-8", timeout: 5_000 },
  );
  return (result.stdout ?? "").trim() === "true";
}

/**
 * Check if Docker is available.
 */
function dockerAvailable(): boolean {
  const result = spawnSync("docker", ["info"], {
    encoding: "utf-8",
    timeout: 5_000,
  });
  return result.status === 0;
}

describe("spwn architect", () => {
  let home: string;
  let originalSpwnHome: string | undefined;
  // We use alpine:latest with a sleep entrypoint for testing
  const testImage = "alpine:latest";

  beforeAll(() => {
    if (!dockerAvailable()) {
      console.warn("Docker not available, skipping architect e2e tests");
      return;
    }

    // Pull alpine image if needed
    if (!imageExists(testImage)) {
      spawnSync("docker", ["pull", testImage], {
        encoding: "utf-8",
        timeout: 60_000,
      });
    }

    // Tag alpine as spwn-architect:latest for testing
    // (The real image runs the architect serve command, but for lifecycle
    // testing we just need a container that stays alive)
    spawnSync(
      "docker",
      [
        "build",
        "-t",
        "spwn-architect:latest",
        "-",
      ],
      {
        input: `FROM alpine:latest\nCMD ["sleep", "infinity"]`,
        encoding: "utf-8",
        timeout: 30_000,
      },
    );
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

  test("status shows not running when no container exists", () => {
    if (!dockerAvailable()) return;

    const result = spwnExec(["architect", "status"], { SPWN_HOME: home });

    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("not running");
  });

  test("start creates and runs a Docker container", () => {
    if (!dockerAvailable()) return;

    const result = spwnExec(["architect", "start"], { SPWN_HOME: home });

    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("Architect started");
    expect(containerRunning(CONTAINER_NAME)).toBe(true);
  });

  test("status shows running after start", () => {
    if (!dockerAvailable()) return;

    // Start first
    spwnExec(["architect", "start"], { SPWN_HOME: home });

    // Then check status
    const result = spwnExec(["architect", "status"], { SPWN_HOME: home });

    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("running");
    expect(result.output).toContain("spwn-architect");
  });

  test("start again shows already running", () => {
    if (!dockerAvailable()) return;

    // Start first time
    spwnExec(["architect", "start"], { SPWN_HOME: home });

    // Start again
    const result = spwnExec(["architect", "start"], { SPWN_HOME: home });

    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("already running");
  });

  test("stop removes the container", () => {
    if (!dockerAvailable()) return;

    // Start first
    spwnExec(["architect", "start"], { SPWN_HOME: home });
    expect(containerRunning(CONTAINER_NAME)).toBe(true);

    // Stop
    const result = spwnExec(["architect", "stop"], { SPWN_HOME: home });

    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("Architect stopped");
    expect(containerRunning(CONTAINER_NAME)).toBe(false);
  });

  test("status shows not running after stop", () => {
    if (!dockerAvailable()) return;

    // Start, then stop
    spwnExec(["architect", "start"], { SPWN_HOME: home });
    spwnExec(["architect", "stop"], { SPWN_HOME: home });

    // Check status
    const result = spwnExec(["architect", "status"], { SPWN_HOME: home });

    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("not running");
  });

  test("stop when not running shows clean error", () => {
    if (!dockerAvailable()) return;

    const result = spwnExec(["architect", "stop"], { SPWN_HOME: home });

    // Should fail with a clean error (not a stack trace)
    expect(result.exitCode).not.toBe(0);
    expect(result.output).toContain("not running");
    expect(result.output).not.toContain("panic");
  });

  test("start with SPWN_ARCHITECT_IMAGE override uses custom image", () => {
    if (!dockerAvailable()) return;

    const result = spwnExec(["architect", "start"], {
      SPWN_HOME: home,
      SPWN_ARCHITECT_IMAGE: "alpine:latest",
    });

    expect(result.exitCode).toBe(0);
    expect(containerRunning(CONTAINER_NAME)).toBe(true);
  });
});
