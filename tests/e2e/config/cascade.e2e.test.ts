import { describe, test, expect, afterEach } from "vitest";
import { existsSync } from "node:fs";
import { join } from "node:path";
import {
  createTestContext,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("config cascade", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("init creates org.yaml", async () => {
    // GIVEN — a fresh SPWN_HOME
    ctx = createTestContext();

    // WHEN — running init
    const result = ctx.spwn(["init"]);

    // THEN — org.yaml exists
    expect(result.exitCode).toBe(0);
    expect(existsSync(join(ctx.home, "org.yaml"))).toBe(true);
  });

  test("init creates default universe config", async () => {
    // GIVEN — a fresh SPWN_HOME
    ctx = createTestContext();

    // WHEN — running init
    const result = ctx.spwn(["init"]);

    // THEN — a default.yaml exists in universes/
    expect(result.exitCode).toBe(0);
    expect(existsSync(join(ctx.home, "universes", "default.yaml"))).toBe(true);
  });

  test("multiple universe configs coexist", async () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init", "alpha"]);
    ctx.spwn(["init", "beta"]);

    // THEN — both configs exist alongside default
    expect(existsSync(join(ctx.home, "universes", "alpha.yaml"))).toBe(true);
    expect(existsSync(join(ctx.home, "universes", "beta.yaml"))).toBe(true);
    expect(existsSync(join(ctx.home, "universes", "default.yaml"))).toBe(true);
  });

  test("named config is used when spawning with -c flag", async () => {
    // GIVEN — a named config
    ctx = createTestContext();
    ctx.spwn(["init", "custom"]);

    // WHEN — spawning with that config
    const spawnResult = ctx.spwn(
      ["universe", "-c", "custom", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — universe ID reflects the config name
    expect(spawnResult.exitCode).toBe(0);
    expect(spawnResult.output).toContain("u-custom-");
  });
});
