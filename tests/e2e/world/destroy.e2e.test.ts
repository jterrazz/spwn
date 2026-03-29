import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("world destroy", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("destroys a running world", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // Verify container is running before destroy
    ctx.universe(id).toBeRunning();

    // WHEN — destroying it
    const destroyResult = ctx.spwn(["world", "destroy", id], 30_000);

    // THEN — exits successfully
    expect(destroyResult.exitCode).toBe(0);
    expect(destroyResult.output).toContain("World destroyed");

    // AND — container no longer exists
    ctx.universe(id).toNotExist();
  });

  test("destroy removes world from list and state", () => {
    // GIVEN — a spawned and destroyed world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // Verify it exists in state before destroy
    ctx.state().hasWorld(id);

    ctx.spwn(["world", "destroy", id], 30_000);

    // WHEN — listing worlds
    const listResult = ctx.spwn(["world", "list"]);

    // THEN — the destroyed world is gone from list
    expect(listResult.exitCode).toBe(0);
    expect(listResult.output).not.toContain(id);

    // AND — gone from state.json
    ctx.state().noWorld(id);
  });

  test("destroy non-existent world fails", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — destroying a world that does not exist
    const result = ctx.spwn(
      ["world", "destroy", "w-nonexistent-00000"],
      30_000,
    );

    // THEN — exits with non-zero code
    expect(result.exitCode).not.toBe(0);
    expect(result.output).toContain("not found");
  });
});
