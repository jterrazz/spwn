import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine, stripAnsi } from "../../setup/output-helpers.js";

describe("visitor", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("visitor without --world flag fails", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — running visitor without world flag
    const result = ctx.spwn(["visitor", "do something"]);

    // THEN — exits with error about required world flag
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /required flag\(s\) "world" not set/);
  });

  test("visitor with non-existent world dispatches (fire-and-forget)", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — running visitor with a non-existent world
    const result = ctx.spwn([
      "visitor",
      "do something",
      "--world",
      "w-nonexistent-00000",
    ]);

    // THEN — dispatches anyway (visitor is fire-and-forget)
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Visitor dispatched: "do something" → w-nonexistent-00000/);
  });

  test("visitor runs ephemeral task in a world", () => {
    // GIVEN — a running world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // Verify container is running before dispatching visitor
    ctx.universe(id).toBeRunning();

    // WHEN — dispatching a visitor task
    const visitorResult = ctx.spwn(
      ["visitor", "lint the code", "--world", id],
      30_000,
    );

    // THEN — succeeds and confirms dispatch with structured output
    expect(visitorResult.exitCode).toBe(0);
    expectLine(visitorResult.output, new RegExp(`Visitor dispatched: "lint the code" → ${id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}`));

    // AND — container is still running after visitor
    ctx.universe(id).toBeRunning();
  });

  test("visitor does not create Mind directory", () => {
    // GIVEN — a running world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — dispatching a visitor task
    ctx.spwn(["visitor", "check health", "--world", id]);

    // THEN — no visitor agent should appear in agent list
    const list = ctx.spwn(["agent", "list"]);
    expect(stripAnsi(list.output)).not.toContain("visitor");
  });
});
