import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";

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

    // THEN — shows the world with agent info
    expect(listResult.exitCode).toBe(0);
    expect(listResult.output).toContain(id);
    expect(listResult.output).toContain("neo");

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

    // THEN — shows world details
    expect(inspectResult.exitCode).toBe(0);
    expect(inspectResult.output).toContain(id);
    expect(inspectResult.output).toContain("docker");
    expect(inspectResult.output).toContain("neo");

    // AND — the container is actually running
    ctx.universe(id).toBeRunning();
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
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // THEN — container is running and has world files
    ctx
      .universe(id)
      .toBeRunning()
      .toHaveFile("/universe/physics.md")
      .toHaveFile("/universe/faculties.md");

    // AND — state tracks it
    ctx.state().hasWorld(id).worldCount(1);

    // AND — inspect works
    const inspectResult = ctx.spwn(["world", "inspect", id]);
    expect(inspectResult.exitCode).toBe(0);
    expect(inspectResult.output).toContain(id);

    // AND — destroy works
    const destroyResult = ctx.spwn(["world", "destroy", id], 30_000);
    expect(destroyResult.exitCode).toBe(0);
    expect(destroyResult.output).toContain("World destroyed");

    // AND — container is gone
    ctx.universe(id).toNotExist();

    // AND — list shows empty
    const listResult = ctx.spwn(["world", "list"]);
    expect(listResult.exitCode).toBe(0);
    expect(listResult.output).not.toContain(id);

    // AND — state no longer has it
    ctx.state().noWorld(id);
  });
});
