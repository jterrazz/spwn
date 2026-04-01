import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";
import { expectLine, lines } from "../../setup/output-helpers.js";

describe("spwn init", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    // GIVEN — a fresh temporary SPWN_HOME directory
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
  });

  test("creates ~/.spwn/ directory structure", async () => {
    // WHEN — running spwn init
    const result = await spwn("init creates directory structure")
      .exec("init")
      .run();

    // THEN — exits successfully and outputs structured status lines
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Created universe\s+org\.yaml/);
    expectLine(result.output, /✓ Created config\s+\w+\.yaml/);
    expectLine(result.output, /✓ Ready\. Next steps:/);
  });

  test("creates org.yaml", async () => {
    // WHEN — running spwn init
    const result = await spwn("init creates org")
      .exec("init")
      .run();

    // THEN — org.yaml creation is confirmed in output
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Created universe\s+org\.yaml/);
  });

  test("creates a world config", async () => {
    // WHEN — running spwn init
    const result = await spwn("init creates config")
      .exec("init")
      .run();

    // THEN — a .yaml config is created (with a random cosmos-themed name)
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Created config\s+\w+\.yaml/);
  });

  test("is idempotent", async () => {
    // GIVEN — init has already been run once
    await spwn("first init").exec("init").run();

    // WHEN — running init again
    const result = await spwn("second init").exec("init").run();

    // THEN — succeeds (creates another config with a different name)
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Created config\s+\w+\.yaml/);
    expectLine(result.output, /✓ Ready\./);
  });

  test("uses random name when none provided", async () => {
    // WHEN — running init without a name argument
    const result = await spwn("init random name")
      .exec("init")
      .run();

    // THEN — a random name is generated and setup completes
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Ready\. Next steps:/);
  });

  test("accepts custom name", async () => {
    // WHEN — running init with a custom name
    const result = await spwn("init with name")
      .exec("init acme")
      .run();

    // THEN — the custom name is used in the config
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Created config\s+acme\.yaml/);
    expectLine(result.output, /✓ Ready\. Next steps:/);
  });
});
