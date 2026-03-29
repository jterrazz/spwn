import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";

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

    // THEN — exits successfully and confirms creation
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("Created");
  });

  test("creates org.yaml", async () => {
    // WHEN — running spwn init
    const result = await spwn("init creates org")
      .exec("init")
      .run();

    // THEN — org.yaml is mentioned in output
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("org.yaml");
  });

  test("creates a world config", async () => {
    // WHEN — running spwn init
    const result = await spwn("init creates config")
      .exec("init")
      .run();

    // THEN — a .yaml config is created (with a random cosmos-themed name)
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain(".yaml");
  });

  test("is idempotent", async () => {
    // GIVEN — init has already been run once
    await spwn("first init").exec("init").run();

    // WHEN — running init again
    const result = await spwn("second init").exec("init").run();

    // THEN — succeeds (creates another config with a different name)
    expect(result.exitCode).toBe(0);
  });

  test("uses random name when none provided", async () => {
    // WHEN — running init without a name argument
    const result = await spwn("init random name")
      .exec("init")
      .run();

    // THEN — a random name is generated and setup completes
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("Ready");
  });

  test("accepts custom name", async () => {
    // WHEN — running init with a custom name
    const result = await spwn("init with name")
      .exec("init acme")
      .run();

    // THEN — the custom name is used in the config
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("acme.yaml");
  });
});
