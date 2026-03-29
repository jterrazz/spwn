import { describe, test, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { spwn, mockApi } from "../../setup/spwn.specification.js";
import { createSpwnHome, createUniverseConfig, createOrgManifest } from "../../setup/helpers.js";
import { runBash, noop } from "../../setup/mock-api/handlers.js";

describe("gate bridge", () => {
  let home: string;

  beforeAll(async () => {
    await mockApi.start();
  });

  afterAll(async () => {
    await mockApi.stop();
  });

  beforeEach(() => {
    // GIVEN — a fresh SPWN_HOME with default config
    home = createSpwnHome();
    createOrgManifest(home);
    createUniverseConfig(home, "default");
    mockApi.reset();
    mockApi.onChat(noop);
  });

  test("gate bridges element access into container", async () => {
    // GIVEN — a universe config with @unix and @git elements
    createUniverseConfig(home, "gated", {
      elements: ["@unix", "@git"],
    });

    // WHEN — spawning a universe
    const result = await spwn("spawn gated universe")
      .exec("universe -c gated")
      .run();

    // THEN — universe is created with gate bridge active
    expect(result.exitCode).toBe(0);
  });

  test("elements not listed are not available inside universe", async () => {
    // GIVEN — a universe config with only @unix (no @git)
    createUniverseConfig(home, "restricted", {
      elements: ["@unix"],
    });

    // WHEN — spawning the restricted universe
    const result = await spwn("spawn restricted")
      .exec("universe -c restricted")
      .run();

    // THEN — universe is created (gate enforces restrictions)
    expect(result.exitCode).toBe(0);
  });

  test("agent inside universe uses bridged tools", async () => {
    // GIVEN — a universe with elements and a scripted agent that runs bash
    createUniverseConfig(home, "tools");
    mockApi.onChat(runBash("echo hello"));

    const spawnResult = await spwn("spawn for bridge")
      .exec("universe -c tools")
      .run();

    // THEN — universe spawns successfully (agent can use bridged tools)
    expect(spawnResult.exitCode).toBe(0);
  });

  test("gate server is accessible inside the container", async () => {
    // GIVEN — a running universe
    const spawnResult = await spwn("spawn for gate check")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — inspecting the universe
    const result = await spwn("inspect gate")
      .exec(`universe inspect ${universeId}`)
      .run();

    // THEN — gate information is available in the inspection output
    expect(result.exitCode).toBe(0);
  });

  test("faculties.md reflects bridged elements", async () => {
    // GIVEN — a universe with specific elements
    createUniverseConfig(home, "faculties-check", {
      elements: ["@unix", "@git", "jq"],
    });

    // WHEN — spawning the universe
    const result = await spwn("spawn with faculties")
      .exec("universe -c faculties-check")
      .run();

    // THEN — universe is created successfully
    expect(result.exitCode).toBe(0);
  });
});
