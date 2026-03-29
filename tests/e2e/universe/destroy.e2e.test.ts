import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("universe destroy", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("destroys a running universe", async () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // WHEN — destroying it
    const destroyResult = ctx.spwn(["universe", "destroy", id], 30_000);

    // THEN — exits successfully
    expect(destroyResult.exitCode).toBe(0);
    expect(destroyResult.output).toContain("Universe destroyed");
  });

  test("destroy removes universe from list", async () => {
    // GIVEN — a spawned and destroyed universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;
    ctx.spwn(["universe", "destroy", id], 30_000);

    // WHEN — listing universes
    const listResult = ctx.spwn(["universe", "list"]);

    // THEN — the destroyed universe is gone
    expect(listResult.exitCode).toBe(0);
    expect(listResult.output).not.toContain(id);
  });

  test("destroy non-existent universe fails", async () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — destroying a universe that does not exist
    const result = ctx.spwn(
      ["universe", "destroy", "u-nonexistent-00000"],
      30_000,
    );

    // THEN — exits with non-zero code
    expect(result.exitCode).not.toBe(0);
    expect(result.output).toContain("not found");
  });
});
