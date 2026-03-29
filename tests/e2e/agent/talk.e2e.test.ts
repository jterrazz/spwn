import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import {
  expectLine,
  expectTableHeader,
  expectTableRow,
} from "../../setup/output-helpers.js";

describe("agent talk", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("talk sees /workspace files", () => {
    // GIVEN — a world with an agent and workspace
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(spawnResult.exitCode).toBe(0);

    // WHEN — talking to the agent about the workspace
    const talkResult = ctx.spwn(
      ["agent", "talk", "neo", "List files in /workspace. Just the filenames, one per line."],
      60_000,
    );

    // THEN — the agent responds with world context
    expect(talkResult.exitCode).toBe(0);
    expectLine(talkResult.output, /neo/);
    expectLine(talkResult.output, /World/);
    // The agent should see files (state.json, worlds/, agents/ etc.)
    expect(talkResult.output.length).toBeGreaterThan(50);
  });

  test("talk can be called multiple times on same world", () => {
    // GIVEN — a world with an agent
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
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
    expectLine(talk1.output, /neo/);
    expectLine(talk2.output, /neo/);
  });

  test("talk to unattached agent fails", () => {
    // GIVEN — an agent exists but is NOT in any world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    ctx.spwn(["agent", "init", "orphan"]);

    // WHEN — trying to talk
    const result = ctx.spwn(["agent", "talk", "orphan", "hello"]);

    // THEN — error about no active world
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /agent "orphan" is not in any active world/);
    expectLine(result.output, /Spawn it first with: spwn world --agent orphan/);
  });

  test("agent list shows world association after spawn", () => {
    // GIVEN — a world with an agent
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — listing agents
    const listResult = ctx.spwn(["agent", "list"]);

    // THEN — agent shows its world and status in table
    expect(listResult.exitCode).toBe(0);
    expectTableHeader(listResult.output, ["NAME", "LAYERS", "WORLD", "STATUS"]);
    expectTableRow(listResult.output, ["neo", id]);
  });

  test("world list shows agent names", () => {
    // GIVEN — a world with an agent
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // WHEN — listing worlds
    const listResult = ctx.spwn(["world", "list"]);

    // THEN — shows agent name in AGENTS column
    expect(listResult.exitCode).toBe(0);
    expectTableHeader(listResult.output, ["ID", "CONFIG", "AGENTS", "STATUS", "CREATED"]);
    expectTableRow(listResult.output, ["neo"]);
  });

  test("talk skips dead containers and finds the live one", () => {
    // GIVEN — a world spawned, destroyed, then respawned
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // Spawn first world
    const spawn1 = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id1 = parseWorldId(spawn1.output)!;

    // Destroy first world
    ctx.spwn(["world", "destroy", id1]);

    // Spawn second world with same agent
    const spawn2 = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id2 = parseWorldId(spawn2.output)!;

    // WHEN — talking to the agent
    const talkResult = ctx.spwn(
      ["agent", "talk", "neo", "hello"],
      60_000,
    );

    // THEN — connects to the live world (id2), not the dead one (id1)
    expect(talkResult.exitCode).toBe(0);
    expectLine(talkResult.output, new RegExp(id2.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));
  });

  test("talk to non-existent agent fails", () => {
    // WHEN — talking to agent that was never created
    ctx = createTestContext();
    const result = ctx.spwn(["agent", "talk", "ghost", "hello"]);

    // THEN — error about agent not found
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /agent "ghost" not found/);
  });

  test("agent list shows unattached after destroy", () => {
    // GIVEN — an agent was in a world, then world destroyed
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // Verify attached first
    const listBefore = ctx.spwn(["agent", "list"]);
    expectTableRow(listBefore.output, [id]);

    // Destroy
    ctx.spwn(["world", "destroy", id]);

    // WHEN — listing agents after destroy
    const listAfter = ctx.spwn(["agent", "list"]);

    // THEN — agent still exists but is unattached
    expectTableRow(listAfter.output, ["neo", "unattached"]);
  });

  test("agent inspect shows world when attached", () => {
    // GIVEN — a world with an agent
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — inspecting the agent
    const inspectResult = ctx.spwn(["agent", "inspect", "neo"]);

    // THEN — shows agent details with Mind layers
    expect(inspectResult.exitCode).toBe(0);
    expectLine(inspectResult.output, /Agent:\s+neo/);
    expectLine(inspectResult.output, /personas\/\s+default\.md/);
  });
});
