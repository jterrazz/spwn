import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("agent talk", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("talk sees /workspace files", () => {
    // GIVEN — a universe with an agent and workspace
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(spawnResult.exitCode).toBe(0);

    // WHEN — talking to the agent about the workspace
    const talkResult = ctx.spwn(
      ["agent", "talk", "neo", "List files in /workspace. Just the filenames, one per line."],
      60_000,
    );

    // THEN — the agent responds (not an error about empty dir)
    expect(talkResult.exitCode).toBe(0);
    expect(talkResult.output).toContain("neo");
    expect(talkResult.output).toContain("Universe");
    // The agent should see files (state.json, universes/, agents/ etc.)
    expect(talkResult.output.length).toBeGreaterThan(50);
  });

  test("talk can be called multiple times on same universe", () => {
    // GIVEN — a universe with an agent
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(spawnResult.exitCode).toBe(0);

    // WHEN — talking twice
    const talk1 = ctx.spwn(
      ["agent", "talk", "neo", "hello"],
      60_000,
    );
    const talk2 = ctx.spwn(
      ["agent", "talk", "neo", "hello again"],
      60_000,
    );

    // THEN — both succeed (agent is still available)
    expect(talk1.exitCode).toBe(0);
    expect(talk2.exitCode).toBe(0);
    expect(talk1.output).toContain("neo");
    expect(talk2.output).toContain("neo");
  });

  test("talk to unattached agent fails", () => {
    // GIVEN — an agent exists but is NOT in any universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    ctx.spwn(["agent", "init", "orphan"]);

    // WHEN — trying to talk
    const result = ctx.spwn(["agent", "talk", "orphan", "hello"]);

    // THEN — error about no active universe
    expect(result.exitCode).not.toBe(0);
    expect(result.output).toContain("not in any active universe");
  });

  test("agent list shows universe association after spawn", () => {
    // GIVEN — a universe with an agent
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // WHEN — listing agents
    const listResult = ctx.spwn(["agent", "list"]);

    // THEN — agent shows its universe and status
    expect(listResult.exitCode).toBe(0);
    expect(listResult.output).toContain("neo");
    expect(listResult.output).toContain(id);
  });

  test("universe list shows agent names", () => {
    // GIVEN — a universe with an agent
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // WHEN — listing universes
    const listResult = ctx.spwn(["universe", "list"]);

    // THEN — shows agent name in AGENTS column
    expect(listResult.exitCode).toBe(0);
    expect(listResult.output).toContain("neo");
    expect(listResult.output).toContain("AGENTS");
  });
});
