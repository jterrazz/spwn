import { describe, test, expect, afterEach } from "vitest";
import { existsSync } from "node:fs";
import { join } from "node:path";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("config cascade", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("init creates org.yaml", () => {
    // GIVEN — a fresh SPWN_HOME
    ctx = createTestContext();

    // WHEN — running init
    const result = ctx.spwn(["init"]);

    // THEN — org.yaml exists
    expect(result.exitCode).toBe(0);
    expect(existsSync(join(ctx.home, "org.yaml"))).toBe(true);
  });

  test("init creates default world config", () => {
    // GIVEN — a fresh SPWN_HOME
    ctx = createTestContext();

    // WHEN — running init
    const result = ctx.spwn(["init"]);

    // THEN — a default.yaml exists in worlds/
    expect(result.exitCode).toBe(0);
    expect(existsSync(join(ctx.home, "worlds", "default.yaml"))).toBe(true);
  });

  test("multiple world configs coexist", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init", "alpha"]);
    ctx.spwn(["init", "beta"]);

    // THEN — both configs exist alongside default
    expect(existsSync(join(ctx.home, "worlds", "alpha.yaml"))).toBe(true);
    expect(existsSync(join(ctx.home, "worlds", "beta.yaml"))).toBe(true);
    expect(existsSync(join(ctx.home, "worlds", "default.yaml"))).toBe(true);
  });

  test("named config is used when spawning with -c flag", () => {
    // GIVEN — a named config
    ctx = createTestContext();
    ctx.spwn(["init", "custom"]);

    // WHEN — spawning with that config
    const spawnResult = ctx.spwn(
      ["world", "-c", "custom", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — world ID reflects the config name
    expect(spawnResult.exitCode).toBe(0);
    expect(spawnResult.output).toContain("w-custom-");

    // AND — container is running
    const id = parseWorldId(spawnResult.output)!;
    ctx.universe(id).toBeRunning();

    // AND — state tracks it with the right config
    ctx.state().hasWorld(id);
  });
});
