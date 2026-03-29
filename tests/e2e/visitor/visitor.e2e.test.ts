import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("visitor", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("visitor without --universe flag fails", async () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — running visitor without universe flag
    const result = ctx.spwn(["visitor", "do something"]);

    // THEN — exits with error mentioning universe requirement
    expect(result.exitCode).not.toBe(0);
    expect(result.output).toContain("universe");
  });

  test("visitor with non-existent universe dispatches (fire-and-forget)", async () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — running visitor with a non-existent universe
    const result = ctx.spwn([
      "visitor",
      "do something",
      "--universe",
      "u-nonexistent-00000",
    ]);

    // THEN — dispatches anyway (visitor is fire-and-forget)
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("Visitor dispatched");
  });

  test("visitor runs ephemeral task in a universe", async () => {
    // GIVEN — a running universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // WHEN — dispatching a visitor task
    const visitorResult = ctx.spwn(
      ["visitor", "lint the code", "--universe", id],
      30_000,
    );

    // THEN — succeeds and confirms dispatch
    expect(visitorResult.exitCode).toBe(0);
    expect(visitorResult.output).toContain("Visitor dispatched");
  });
});
