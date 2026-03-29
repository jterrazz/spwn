import { describe, test, expect, afterEach } from "vitest";
import { existsSync } from "node:fs";
import { join } from "node:path";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine } from "../../setup/output-helpers.js";

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

    // THEN — org.yaml exists and output confirms creation
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Created organization\s+org\.yaml/);
    expect(existsSync(join(ctx.home, "org.yaml"))).toBe(true);
  });

  test("init creates default world config", () => {
    // GIVEN — a fresh SPWN_HOME
    ctx = createTestContext();

    // WHEN — running init
    const result = ctx.spwn(["init"]);

    // THEN — a default.yaml exists in worlds/ and output confirms
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Created config\s+\w+\.yaml/);
    expect(existsSync(join(ctx.home, "worlds", "default.yaml"))).toBe(true);
  });

  test("multiple world configs coexist", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    const r1 = ctx.spwn(["init", "alpha"]);
    const r2 = ctx.spwn(["init", "beta"]);

    // THEN — init outputs confirm named configs
    expectLine(r1.output, /✓ Created config\s+alpha\.yaml/);
    expectLine(r2.output, /✓ Created config\s+beta\.yaml/);

    // AND — both configs exist alongside default
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
    expectLine(spawnResult.output, /✓ Spawned world\s+w-custom-\d{5}/);

    // AND — container is running
    const id = parseWorldId(spawnResult.output)!;
    ctx.universe(id).toBeRunning();

    // AND — state tracks it with the right config
    ctx.state().hasWorld(id);
  });
});
