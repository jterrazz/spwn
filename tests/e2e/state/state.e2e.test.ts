import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("state management", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("world state persists in state.json", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // THEN — state.json exists and contains the world
    ctx
      .state()
      .exists()
      .hasWorld(id)
      .hasAgent(id, "neo")
      .worldCount(1);

    // AND — the container is actually running
    ctx.universe(id).toBeRunning();
  });

  test("destroy updates state file", () => {
    // GIVEN — a spawned and destroyed world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // Verify state before destroy
    ctx.state().hasWorld(id);

    ctx.spwn(["world", "destroy", id], 30_000);

    // THEN — state.json no longer contains the world
    ctx.state().noWorld(id);

    // AND — container is gone
    ctx.universe(id).toNotExist();
  });

  test("state tracks active worlds across list calls", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — listing worlds multiple times
    const list1 = ctx.spwn(["world", "list"]);
    const list2 = ctx.spwn(["world", "list"]);

    // THEN — both calls show the same world
    expect(list1.output).toContain(id);
    expect(list2.output).toContain(id);

    // AND — state is consistent
    ctx.state().hasWorld(id).worldCount(1);
  });

  test("multiple worlds tracked in state", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // Spawn two worlds
    const r1 = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id1 = parseWorldId(r1.output)!;

    const r2 = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id2 = parseWorldId(r2.output)!;

    // THEN — both are tracked
    ctx
      .state()
      .worldCount(2)
      .hasWorld(id1)
      .hasWorld(id2);

    // AND — both containers are running
    ctx.universe(id1).toBeRunning();
    ctx.universe(id2).toBeRunning();
  });
});
