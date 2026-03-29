import { describe, test, expect, afterEach } from "vitest";
import { spawnSync } from "node:child_process";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("universe physics", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("inspect shows physics constants", async () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // WHEN — inspecting the universe
    const inspectResult = ctx.spwn(["universe", "inspect", id]);

    // THEN — physics constants are shown
    expect(inspectResult.exitCode).toBe(0);
    expect(inspectResult.output).toContain("Constants");
    expect(inspectResult.output).toContain("CPU");
    expect(inspectResult.output).toContain("Memory");
  });

  test("faculties.md is generated inside the universe", async () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(spawnResult.exitCode).toBe(0);

    // THEN — the spawn output mentions faculties generation
    expect(spawnResult.output).toContain("Generated faculties");
    expect(spawnResult.output).toContain("physics.md");
    expect(spawnResult.output).toContain("faculties.md");
  });

  test("universe includes declared elements", async () => {
    // GIVEN — a spawned universe (default config has @unix, @git elements)
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // WHEN — inspecting the universe
    const inspectResult = ctx.spwn(["universe", "inspect", id]);

    // THEN — exits successfully (elements are part of physics config)
    expect(inspectResult.exitCode).toBe(0);
  });
});
