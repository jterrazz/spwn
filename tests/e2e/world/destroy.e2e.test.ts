import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine, expectNoLine } from "../../setup/output-helpers.js";

describe("down", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("destroys a running world", () => {
    // GIVEN - a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // Verify container is running before destroy
    ctx.world(id).toBeRunning();

    // WHEN - destroying it
    const destroyResult = ctx.spwn(["down", id], 30_000);

    // THEN - exits successfully with structured status lines
    expect(destroyResult.exitCode).toBe(0);
    expectLine(destroyResult.output, /→ Destroying world\.\.\./);
    expectLine(destroyResult.output, /✓ Stopped agent/);
    expectLine(destroyResult.output, /✓ Removed container/);
    expectLine(destroyResult.output, /✓ Mind persisted/);
    expectLine(destroyResult.output, /✓ World destroyed\. Agent survives\./);

    // AND - container no longer exists
    ctx.world(id).toNotExist();
  });

  test("destroy removes world from list and state", () => {
    // GIVEN - a spawned and destroyed world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // Verify it exists in state before destroy
    ctx.state().hasWorld(id);

    ctx.spwn(["down", id], 30_000);

    // WHEN - listing worlds
    const listResult = ctx.spwn(["ls"]);

    // THEN - the destroyed world is gone from list
    expect(listResult.exitCode).toBe(0);
    expectNoLine(listResult.output, new RegExp(id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));

    // AND - gone from state.json
    ctx.state().noWorld(id);
  });

  test("destroy non-existent world fails", () => {
    // GIVEN - an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN - destroying a world that does not exist
    const result = ctx.spwn(
      ["down", "w-nonexistent-00000"],
      30_000,
    );

    // THEN - exits with non-zero code and structured error
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /✗ Destroy failed\s+world w-nonexistent-00000 not found/);
  });
});
