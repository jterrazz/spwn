import { describe, test, expect, beforeEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";

describe("spwn init", () => {
  let home: string;

  beforeEach(() => {
    // GIVEN — a fresh temporary SPWN_HOME directory
    home = createSpwnHome();
  });

  test("creates ~/.spwn/ directory structure", async () => {
    // WHEN — running spwn init
    const result = await spwn("init creates directory structure")
      .exec("init")
      .run();

    // THEN — exits successfully and confirms creation
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("Created");
  });

  test("creates org.yaml", async () => {
    // WHEN — running spwn init
    const result = await spwn("init creates org")
      .exec("init")
      .run();

    // THEN — org.yaml is mentioned in output
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("org.yaml");
  });

  test("creates default.yaml universe config", async () => {
    // WHEN — running spwn init
    const result = await spwn("init creates default config")
      .exec("init")
      .run();

    // THEN — default.yaml is created
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("default.yaml");
  });

  test("is idempotent", async () => {
    // GIVEN — init has already been run once
    await spwn("first init").exec("init").run();

    // WHEN — running init again
    const result = await spwn("second init").exec("init").run();

    // THEN — succeeds and reports already exists
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("already exists");
  });

  test("uses random name when none provided", async () => {
    // WHEN — running init without a name argument
    const result = await spwn("init random name")
      .exec("init")
      .run();

    // THEN — a random name is generated and setup completes
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("Ready");
  });

  test("accepts custom name", async () => {
    // WHEN — running init with a custom org name
    const result = await spwn("init with name")
      .exec("init acme")
      .run();

    // THEN — the custom name is used in the config
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("acme.yaml");
  });
});
