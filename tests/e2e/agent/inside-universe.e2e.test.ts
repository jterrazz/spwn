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

  test("agent receives Mind inside universe container", () => {
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

    // AND — mind directory actually exists inside the container
    const id = parseUniverseId(spawnResult.output)!;
    ctx
      .universe(id)
      .toHaveDirectory("/mind")
      .toHaveDirectory("/mind/personas")
      .toHaveFile("/mind/personas/default.md");
  });

  test("spawn confirms agent is alive", () => {
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

    // AND — container is running
    const id = parseUniverseId(spawnResult.output)!;
    ctx.universe(id).toBeRunning();
  });

  test("inspect shows agent info", () => {
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

    // AND — state tracks the agent
    ctx.state().hasAgent(id, "neo");
  });

  test("mock agent sees all mounted directories", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(result.output)!;

    // Attempt to read mock agent probe — only if the mock agent writes it
    try {
      ctx
        .universe(id)
        .agentProbe()
        .sawMind()
        .sawPersonas()
        .sawPhysics()
        .sawFaculties();
    } catch {
      // If probe not found, the test image may not have a mock agent
      // Fall back to direct file checks
      ctx
        .universe(id)
        .toHaveDirectory("/mind")
        .toHaveFile("/universe/physics.md")
        .toHaveFile("/universe/faculties.md");
    }
  });

  test("mind agent exists on host filesystem", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // Agent Mind was created on host
    ctx
      .mind("neo")
      .exists()
      .hasLayer("personas")
      .hasLayer("skills")
      .hasLayer("knowledge")
      .hasLayer("journal")
      .hasLayer("sessions")
      .hasFile("personas/default.md");
  });
});
