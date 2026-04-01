import { describe, test, expect, afterEach } from "vitest";
import { writeFileSync, existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("world workspace persistence", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("host files are visible inside container", () => {
    // GIVEN — a workspace with a test file
    ctx = createTestContext();
    ctx.spwn(["init"]);
    writeFileSync(join(ctx.home, "host-file.txt"), "created on host");

    // WHEN — spawning a world with that workspace
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;

    // THEN — the file exists inside the container at /workspace/
    ctx.universe(id).toHaveFile("/workspace/host-file.txt", "created on host");
  });

  test("container changes persist to host", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;

    // WHEN — creating a file inside the container
    ctx.universe(id).exec("echo 'created in container' > /workspace/container-file.txt");

    // THEN — the file exists on the host filesystem
    const hostPath = join(ctx.home, "container-file.txt");
    expect(existsSync(hostPath)).toBe(true);
    const content = readFileSync(hostPath, "utf-8").trim();
    expect(content).toBe("created in container");
  });

  test("workspace mount is read-write", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;

    // WHEN — writing and reading back inside container
    ctx.universe(id).exec("echo 'rw-test-content' > /workspace/rw-test.txt");
    const readBack = ctx.universe(id).exec("cat /workspace/rw-test.txt");

    // THEN — content matches
    expect(readBack).toBe("rw-test-content");
  });

  test("mock agent output file exists in workspace", () => {
    // GIVEN — a spawned world with mock agent
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;

    // THEN — mock-claude wrote its output file inside the container
    const mockOutput = ctx.universe(id).readFile("/workspace/mock-output.txt");
    expect(mockOutput).toContain("mock-claude was here");
  });

  test("spawn with non-existent workspace path fails gracefully", () => {
    // GIVEN — an initialized context
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning with a workspace path that doesn't exist
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", "/tmp/nonexistent-path-12345"],
      60_000,
    );

    // THEN — exits with error (no stack trace)
    expect(result.exitCode).not.toBe(0);
    expect(result.output).not.toContain("TypeError");
    expect(result.output).not.toContain("FATAL");
  });
});
