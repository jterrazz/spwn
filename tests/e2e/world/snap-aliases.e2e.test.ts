import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine, stripAnsi } from "../../setup/output-helpers.js";

describe("snap command aliases", () => {
  let ctx: TestContext;

  afterEach(() => {
    // Clean up snapshot images
    try {
      const { spawnSync } = require("node:child_process");
      const images = spawnSync(
        "docker",
        [
          "images",
          "--format",
          "{{.Repository}}:{{.Tag}}",
          "--filter",
          "reference=spwn-snapshot:*",
        ],
        { encoding: "utf-8" },
      );
      for (const tag of (images.stdout || "")
        .trim()
        .split("\n")
        .filter(Boolean)) {
        spawnSync("docker", ["rmi", "-f", tag], { encoding: "utf-8" });
      }
    } catch {}
    ctx?.cleanup();
  });

  test("'spwn snap --help' shows subcommands", () => {
    ctx = createTestContext();
    const result = ctx.spwn(["snap", "--help"]);

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("save");
    expect(out).toContain("ls");
    expect(out).toContain("restore");
    expect(out).toContain("rm");
  });

  test("'spwn snap save <id>' creates a snapshot", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // WHEN — using the 'snap save' alias
    const result = ctx.spwn(["snap", "save", id, "--name", "alias-test"]);

    // THEN — snapshot created
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /[Ss]napshot|[Ss]aved/);
  });

  test("'spwn snap ls' lists snapshots", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    ctx.spwn(["snap", "save", id, "--name", "ls-test"]);

    // WHEN — listing snapshots via alias
    const result = ctx.spwn(["snap", "ls"]);

    // THEN — shows the snapshot
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("ls-test");
  });

  test("'spwn snap rm' removes a snapshot", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    ctx.spwn(["snap", "save", id, "--name", "rm-test"]);

    // WHEN — removing via alias
    const result = ctx.spwn(["snap", "rm", id + "--rm-test"]);

    // THEN — succeeds
    expect(result.exitCode).toBe(0);

    // AND — no longer in list
    const list = ctx.spwn(["snap", "ls"]);
    expect(stripAnsi(list.output)).not.toContain("rm-test");
  });

  test("'spwn snap restore' restores from snapshot", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--no-agent", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    ctx.spwn(["snap", "save", id, "--name", "restore-alias"]);

    // Destroy original
    ctx.spwn(["world", "destroy", id]);

    // WHEN — restoring via alias
    const result = ctx.spwn(
      ["snap", "restore", id + "--restore-alias"],
      60_000,
    );

    // THEN — new world created
    expect(result.exitCode).toBe(0);
    const newId = parseWorldId(result.output);
    expect(newId).toBeTruthy();
    expect(newId).not.toBe(id);
  });
});
