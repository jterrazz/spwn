import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine, stripAnsi } from "../../setup/output-helpers.js";

describe("agent --npc", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("npc without --world flag fails", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — running agent --npc without world flag
    const result = ctx.spwn(["agent", "--npc", "do something"]);

    // THEN — exits with error about required world flag
    expect(result.exitCode).not.toBe(0);
  });

  test("npc dispatches task in world", () => {
    // GIVEN — a running world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // Verify container is running before dispatching NPC
    ctx.universe(id).toBeRunning();

    // WHEN — dispatching an NPC task
    const npcResult = ctx.spwn(
      ["agent", "--npc", "lint the code", "--world", id],
      30_000,
    );

    // THEN — succeeds
    expect(npcResult.exitCode).toBe(0);

    // AND — container is still running after NPC
    ctx.universe(id).toBeRunning();
  });

  test("npc does not create Mind directory", () => {
    // GIVEN — a running world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — dispatching an NPC task
    ctx.spwn(["agent", "--npc", "check health", "--world", id]);

    // THEN — no NPC agent should appear in agent ls
    const list = ctx.spwn(["agent", "ls"]);
    expect(stripAnsi(list.output)).not.toContain("npc");
  });
});
