import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";

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

    // THEN — exits with error mentioning world requirement
    expect(result.exitCode).not.toBe(0);
    expect(result.output).toContain("world");
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
    expect(result.output).toContain("Visitor dispatched");
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

    // THEN — succeeds and confirms dispatch
    expect(visitorResult.exitCode).toBe(0);
    expect(visitorResult.output).toContain("Visitor dispatched");

    // AND — container is still running after visitor
    ctx.universe(id).toBeRunning();
  });
});
