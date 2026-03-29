import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("state management", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("universe state persists in state.json", () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // THEN — state.json exists and contains the universe
    ctx
      .state()
      .exists()
      .hasUniverse(id)
      .hasAgent(id, "neo")
      .universeCount(1);

    // AND — the container is actually running
    ctx.universe(id).toBeRunning();
  });

  test("destroy updates state file", () => {
    // GIVEN — a spawned and destroyed universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // Verify state before destroy
    ctx.state().hasUniverse(id);

    ctx.spwn(["universe", "destroy", id], 30_000);

    // THEN — state.json no longer contains the universe
    ctx.state().noUniverse(id);

    // AND — container is gone
    ctx.universe(id).toNotExist();
  });

  test("state tracks active universes across list calls", () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // WHEN — listing universes multiple times
    const list1 = ctx.spwn(["universe", "list"]);
    const list2 = ctx.spwn(["universe", "list"]);

    // THEN — both calls show the same universe
    expect(list1.output).toContain(id);
    expect(list2.output).toContain(id);

    // AND — state is consistent
    ctx.state().hasUniverse(id).universeCount(1);
  });

  test("multiple universes tracked in state", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // Spawn two universes
    const r1 = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id1 = parseUniverseId(r1.output)!;

    const r2 = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id2 = parseUniverseId(r2.output)!;

    // THEN — both are tracked
    ctx
      .state()
      .universeCount(2)
      .hasUniverse(id1)
      .hasUniverse(id2);

    // AND — both containers are running
    ctx.universe(id1).toBeRunning();
    ctx.universe(id2).toBeRunning();
  });
});
