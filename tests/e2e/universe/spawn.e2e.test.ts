import { describe, test, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { spwn, mockApi } from "../../setup/spwn.specification.js";
import { createSpwnHome, createUniverseConfig, createOrgManifest } from "../../setup/helpers.js";
import { noop } from "../../setup/mock-api/handlers.js";

describe("universe spawn", () => {
  let home: string;

  beforeAll(async () => {
    await mockApi.start();
  });

  afterAll(async () => {
    await mockApi.stop();
  });

  beforeEach(() => {
    // GIVEN — a fresh SPWN_HOME with org manifest and default universe config
    home = createSpwnHome();
    createOrgManifest(home);
    createUniverseConfig(home, "default");
    mockApi.reset();
    mockApi.onChat(noop);
  });

  test("spawns a universe with default config", async () => {
    // WHEN — running bare spwn universe
    const result = await spwn("spawn default universe")
      .exec("universe")
      .run();

    // THEN — universe is created with an ID matching u-default-XXXXX
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toMatch(/u-default-\d{5}/);
  });

  test("spawns a universe with named config via -c flag", async () => {
    // GIVEN — an additional universe config named "acme"
    createUniverseConfig(home, "acme");

    // WHEN — spawning with -c acme
    const result = await spwn("spawn named universe")
      .exec("universe -c acme")
      .run();

    // THEN — universe ID contains the config name
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toMatch(/u-acme-\d{5}/);
  });

  test("spawns a universe with governor agent", async () => {
    // WHEN — spawning with --governor flag
    const result = await spwn("spawn with governor")
      .exec("universe --governor morpheus")
      .run();

    // THEN — universe is created with a governor
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("morpheus");
  });

  test("spawns a universe with workspace mount", async () => {
    // WHEN — spawning with -w flag pointing to sample project
    const result = await spwn("spawn with workspace")
      .exec("universe -w ./sample-project")
      .run();

    // THEN — workspace is mounted into the universe
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("sample-project");
  });

  test("fails with non-existent config", async () => {
    // WHEN — spawning with a config that does not exist
    const result = await spwn("spawn missing config")
      .exec("universe -c nonexistent")
      .run();

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
    expect(result.stderr).toContain("not found");
  });

  test("universe ID format is u-{name}-{5digits}", async () => {
    // WHEN — spawning a universe
    const result = await spwn("spawn check ID format")
      .exec("universe")
      .run();

    // THEN — stdout contains a properly formatted universe ID
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toMatch(/u-[a-z]+-\d{5}/);
  });
});
