import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine, stripAnsi } from "../../setup/output-helpers.js";

describe("world snapshots", () => {
  let ctx: TestContext;

  afterEach(() => {
    // Clean up snapshot images (global Docker state, not per SPWN_HOME)
    try {
      const { spawnSync } = require("node:child_process");
      const images = spawnSync("docker", ["images", "--format", "{{.Repository}}:{{.Tag}}", "--filter", "reference=spwn-snapshot:*"], { encoding: "utf-8" });
      for (const tag of (images.stdout || "").trim().split("\n").filter(Boolean)) {
        spawnSync("docker", ["rmi", "-f", tag], { encoding: "utf-8" });
      }
    } catch {}
    ctx?.cleanup();
  });

  test("snapshot saves a running world", () => {
    // GIVEN — a running world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(["world", "--agent", "neo", "-w", ctx.home], 60_000);
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — taking a snapshot
    const result = ctx.spwn(["world", "snapshot", id, "--name", "test-snap"]);

    // THEN — succeeds with snapshot tag
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Saved snapshot/);
    expect(stripAnsi(result.output)).toContain("test-snap");
  });

  test("snapshots lists saved snapshots", () => {
    // GIVEN — a world with a snapshot
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(["world", "--agent", "neo", "-w", ctx.home], 60_000);
    const id = parseWorldId(spawnResult.output)!;
    ctx.spwn(["world", "snapshot", id, "--name", "my-snap"]);

    // WHEN — listing snapshots
    const result = ctx.spwn(["world", "snapshots"]);

    // THEN — shows the snapshot
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("my-snap");
    expect(stripAnsi(result.output)).toContain(id);
  });

  test("restore creates new world from snapshot", () => {
    // GIVEN — a snapshot exists
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(["world", "--no-agent", "-w", ctx.home], 60_000);
    const id = parseWorldId(spawnResult.output)!;
    ctx.spwn(["world", "snapshot", id, "--name", "restore-test"]);

    // Destroy original
    ctx.spwn(["world", "destroy", id]);

    // WHEN — restoring from snapshot (no agent, just the world)
    const result = ctx.spwn(
      ["world", "restore", id + "--restore-test"],
      60_000,
    );

    // THEN — new world created
    expect(result.exitCode).toBe(0);
    const newId = parseWorldId(result.output);
    expect(newId).toBeTruthy();
    expect(newId).not.toBe(id);
    expectLine(result.output, /Restored world/);
  });

  test("snapshot delete removes snapshot", () => {
    // GIVEN — a snapshot exists
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(["world", "--agent", "neo", "-w", ctx.home], 60_000);
    const id = parseWorldId(spawnResult.output)!;
    ctx.spwn(["world", "snapshot", id, "--name", "to-delete"]);

    // WHEN — deleting the snapshot
    const result = ctx.spwn(["world", "snapshot", "delete", id + "--to-delete"]);

    // THEN — succeeds
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Deleted snapshot/);

    // AND — no longer in list
    const list = ctx.spwn(["world", "snapshots"]);
    expect(stripAnsi(list.output)).not.toContain("to-delete");
  });

  test("snapshot non-existent world fails", () => {
    ctx = createTestContext();
    const result = ctx.spwn(["world", "snapshot", "w-nonexistent-00000"]);
    expect(result.exitCode).not.toBe(0);
    expect(stripAnsi(result.output)).toContain("w-nonexistent-00000");
  });

  test("snapshots empty list", () => {
    ctx = createTestContext();
    const result = ctx.spwn(["world", "snapshots"]);
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("No snapshots");
  });
});
