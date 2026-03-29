import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("agent inside universe", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("agent receives Mind inside universe container", async () => {
    // GIVEN — a spawned universe with agent neo
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — the output confirms mind was mounted
    expect(spawnResult.exitCode).toBe(0);
    expect(spawnResult.output).toContain("Mounted mind");
    expect(spawnResult.output).toContain("neo");
    expect(spawnResult.output).toContain("/mind");
  });

  test("spawn confirms agent is alive", async () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — agent is reported as alive
    expect(spawnResult.exitCode).toBe(0);
    expect(spawnResult.output).toContain("Agent is alive");
  });

  test("inspect shows agent info", async () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // WHEN — inspecting
    const inspectResult = ctx.spwn(["universe", "inspect", id]);

    // THEN — agent is shown
    expect(inspectResult.exitCode).toBe(0);
    expect(inspectResult.output).toContain("Agent");
    expect(inspectResult.output).toContain("neo");
  });
});
