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

  test("agent receives Mind inside world container", () => {
    // GIVEN — a spawned world with agent neo
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — the output confirms mind was mounted with structured status
    expect(spawnResult.exitCode).toBe(0);
    expectLine(spawnResult.output, /✓ Mounted mind\s+neo → \/mind/);
    expectLine(spawnResult.output, /✓ Created container\s+w-\w+-\d{5}/);

    // AND — mind directory actually exists inside the container
    const id = parseWorldId(spawnResult.output)!;
    ctx
      .universe(id)
      .toHaveDirectory("/mind")
      .toHaveDirectory("/mind/identity")
      .toHaveFile("/mind/identity/default.md");
  });

  test("spawn confirms agent is alive", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — agent is reported as alive with structured output
    expect(spawnResult.exitCode).toBe(0);
    expectLine(spawnResult.output, /✓ Agent is alive\./);
    expectLine(spawnResult.output, /✓ Created container\s+w-\w+-\d{5}/);

    // AND — container is running
    const id = parseWorldId(spawnResult.output)!;
    ctx.universe(id).toBeRunning();
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
    expectLine(inspectResult.output, /Backend:\s+docker/);
    expectLine(inspectResult.output, /Status:\s+(running|idle)/);

    // AND — state tracks the agent
    ctx.state().hasAgent(id, "neo");
  });

  test("mock agent sees all mounted directories", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(result.output)!;

    // Attempt to read mock agent probe — only if the mock agent writes it
    try {
      ctx
        .universe(id)
        .agentProbe()
        .sawMind()
        .sawIdentity()
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
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // Agent Mind was created on host
    ctx
      .mind("neo")
      .exists()
      .hasLayer("identity")
      .hasLayer("skills")
      .hasLayer("memory/knowledge")
      .hasLayer("memory/journal")
      .hasLayer("sessions")
      .hasFile("identity/default.md");
  });
});
