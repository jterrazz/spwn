import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("universe lifecycle", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("list shows spawned universes", () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // WHEN — listing universes
    const listResult = ctx.spwn(["universe", "list"]);

    // THEN — shows the universe with agent info
    expect(listResult.exitCode).toBe(0);
    expect(listResult.output).toContain(id);
    expect(listResult.output).toContain("neo");

    // AND — state tracks it with running status
    ctx.state().hasUniverse(id).hasAgent(id, "neo");
  });

  test("inspect shows universe details", () => {
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

    // THEN — shows universe details
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
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(spawnResult.exitCode).toBe(0);
    const id = parseUniverseId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // THEN — container is running and has universe files
    ctx
      .universe(id)
      .toBeRunning()
      .toHaveFile("/universe/physics.md")
      .toHaveFile("/universe/faculties.md");

    // AND — state tracks it
    ctx.state().hasUniverse(id).universeCount(1);

    // AND — inspect works
    const inspectResult = ctx.spwn(["universe", "inspect", id]);
    expect(inspectResult.exitCode).toBe(0);
    expect(inspectResult.output).toContain(id);

    // AND — destroy works
    const destroyResult = ctx.spwn(["universe", "destroy", id], 30_000);
    expect(destroyResult.exitCode).toBe(0);
    expect(destroyResult.output).toContain("Universe destroyed");

    // AND — container is gone
    ctx.universe(id).toNotExist();

    // AND — list shows empty
    const listResult = ctx.spwn(["universe", "list"]);
    expect(listResult.exitCode).toBe(0);
    expect(listResult.output).not.toContain(id);

    // AND — state no longer has it
    ctx.state().noUniverse(id);
  });
});
