import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import {
  expectLine,
  expectNoLine,
  expectTableHeader,
  expectTableRow,
} from "../../setup/output-helpers.js";

describe("world lifecycle", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("list shows spawned worlds", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // WHEN — listing worlds
    const listResult = ctx.spwn(["world", "list"]);

    // THEN — shows the world with agent info in a table
    expect(listResult.exitCode).toBe(0);
    expectTableHeader(listResult.output, ["ID", "CONFIG", "AGENTS", "STATUS", "CREATED"]);
    expectTableRow(listResult.output, [id, "default", "neo"]);

    // AND — state tracks it with running status
    ctx.state().hasWorld(id).hasAgent(id, "neo");
  });

  test("inspect shows world details", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — inspecting the world
    const inspectResult = ctx.spwn(["world", "inspect", id]);

    // THEN — shows world details in structured format
    expect(inspectResult.exitCode).toBe(0);
    expectLine(inspectResult.output, new RegExp(`World:\\s+${id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}`));
    expectLine(inspectResult.output, /Config:\s+default/);
    expectLine(inspectResult.output, /Backend:\s+docker/);
    expectLine(inspectResult.output, /Status:\s+(running|idle)/);
    expectLine(inspectResult.output, /Agent:\s+a-neo-\d{5}/);
    expectLine(inspectResult.output, /Constants:/);
    expectLine(inspectResult.output, /Workspaces:/);
    expectLine(inspectResult.output, /Agent home:/);

    // AND — the container is actually running
    ctx.world(id).toBeRunning();
  });

  test("full lifecycle: spawn, inspect, destroy", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawn
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(spawnResult.exitCode).toBe(0);
    expectLine(spawnResult.output, /✓ Created container\s+(?:spwn-world|w)-\w+-\d{5}/);
    expectLine(spawnResult.output, /✓ Agent is alive\./);
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // THEN — container is running and has world files
    ctx
      .universe(id)
      .toBeRunning()
      .toHaveFile("/world/physics.md")
      .toHaveFile("/world/faculties.md");

    // AND — state tracks it
    ctx.state().hasWorld(id).worldCount(1);

    // AND — inspect works with structured output
    const inspectResult = ctx.spwn(["world", "inspect", id]);
    expect(inspectResult.exitCode).toBe(0);
    expectLine(inspectResult.output, new RegExp(`World:\\s+${id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}`));
    expectLine(inspectResult.output, /Status:\s+(running|idle)/);

    // AND — destroy works with structured status
    const destroyResult = ctx.spwn(["world", "destroy", id], 30_000);
    expect(destroyResult.exitCode).toBe(0);
    expectLine(destroyResult.output, /✓ World destroyed\. Agent survives\./);

    // AND — container is gone
    ctx.world(id).toNotExist();

    // AND — list shows empty
    const listResult = ctx.spwn(["world", "list"]);
    expect(listResult.exitCode).toBe(0);
    expectNoLine(listResult.output, new RegExp(id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));

    // AND — state no longer has it
    ctx.state().noWorld(id);
  });
});
