import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine } from "../../setup/output-helpers.js";

describe("agent inside world", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("agent is mounted at /agents/<name> inside the world container", () => {
    // GIVEN — a spawned world with agent neo
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    expect(spawnResult.exitCode).toBe(0);
    expectLine(spawnResult.output, /✓ Created container\s+(?:spwn-world|w)-\w+-\d{5}/);

    // The agent's home dir is mounted via the single /agents bind.
    const id = parseWorldId(spawnResult.output)!;
    ctx
      .universe(id)
      .toHaveDirectory("/agents/neo")
      .toHaveDirectory("/agents/neo/core")
      .toHaveFile("/agents/neo/core/profile.md");
  });

  test("spawn confirms agent is alive", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    expect(spawnResult.exitCode).toBe(0);
    expectLine(spawnResult.output, /✓ Agent is alive\./);
    expectLine(spawnResult.output, /✓ Created container\s+(?:spwn-world|w)-\w+-\d{5}/);

    const id = parseWorldId(spawnResult.output)!;
    ctx.world(id).toBeRunning();
  });

  test("inspect shows agent info", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — inspecting
    const inspectResult = ctx.spwn(["world", "inspect", id]);

    // THEN — agent is shown with structured details
    expect(inspectResult.exitCode).toBe(0);
    expectLine(inspectResult.output, /World:\s+/);
    expectLine(inspectResult.output, /Agent:\s+a-neo-\d{5}/);
    expectLine(inspectResult.output, /Status:\s+(running|idle)/);

    // AND — labels track the agent
    ctx.state().hasAgent(id, "neo");
  });

  test("mind agent exists on host filesystem", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // Agent Mind was created on host
    ctx
      .mind("neo")
      .exists()
      .hasLayer("core")
      .hasLayer("skills")
      .hasLayer("knowledge")
      .hasLayer("journal")
      .hasFile("core/profile.md");
  });
});
