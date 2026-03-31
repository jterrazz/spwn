import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine, stripAnsi } from "../../setup/output-helpers.js";

describe("error recovery — Docker", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("double destroy — second destroy fails gracefully", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();
    ctx.universe(id).toBeRunning();

    // WHEN — destroying it the first time
    const firstDestroy = ctx.spwn(["down", id], 30_000);
    expect(firstDestroy.exitCode).toBe(0);

    // AND — destroying it a second time
    const secondDestroy = ctx.spwn(["down", id], 30_000);

    // THEN — second destroy fails with clean error (not a crash)
    expect(secondDestroy.exitCode).not.toBe(0);
    expectLine(secondDestroy.output, /not found/);

    // AND — no usage dump on error
    const output = stripAnsi(secondDestroy.output);
    expect(output).not.toContain("Available Commands:");
    expect(output).not.toContain("Global Flags:");
  });

  test("inspect non-existent world — clean error", () => {
    // GIVEN — an initialized context
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — inspecting a world that was never created
    const result = ctx.spwn(["world", "inspect", "w-ghost-99999"]);

    // THEN — exits with clean error
    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    expect(output).toMatch(/not found/);
    expect(output).not.toContain("Available Commands:");
  });

  test("spawn with invalid config — no container leaked", () => {
    // GIVEN — an initialized context
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning with a non-existent config
    const result = ctx.spwn(
      ["world", "-c", "nonexistent", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);

    // AND — no worlds are listed (no leaked container)
    const listResult = ctx.spwn(["ls"]);
    expect(listResult.exitCode).toBe(0);
    const worldIds = stripAnsi(listResult.output).match(/w-nonexistent-\d{5}/g);
    expect(worldIds).toBeNull();
  });

  test("after error, next command still works — no corrupted state", () => {
    // GIVEN — an initialized context
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — triggering an error (destroy non-existent)
    const errorResult = ctx.spwn(["down", "w-ghost-00000"], 30_000);
    expect(errorResult.exitCode).not.toBe(0);

    // AND — running a normal command after the error
    const statusResult = ctx.spwn(["world", "list"]);

    // THEN — the next command works fine (state not corrupted)
    expect(statusResult.exitCode).toBe(0);
  });

  test("spawn and destroy cycle — world can be fully recreated after destroy", () => {
    // GIVEN — an initialized context
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawn, destroy, spawn again
    const first = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(first.exitCode).toBe(0);
    const firstId = parseWorldId(first.output)!;
    expect(firstId).toBeTruthy();

    ctx.spwn(["down", firstId], 30_000);

    const second = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — second spawn succeeds with a different world ID
    expect(second.exitCode).toBe(0);
    const secondId = parseWorldId(second.output)!;
    expect(secondId).toBeTruthy();
    expect(secondId).not.toBe(firstId);
    ctx.universe(secondId).toBeRunning();
  });
});
