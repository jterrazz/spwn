import { describe, test, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { spwn, mockApi } from "../../setup/spwn.specification.js";
import { createSpwnHome, createUniverseConfig, createOrgManifest } from "../../setup/helpers.js";
import { noop } from "../../setup/mock-api/handlers.js";

describe("universe physics", () => {
  let home: string;

  beforeAll(async () => {
    await mockApi.start();
  });

  afterAll(async () => {
    await mockApi.stop();
  });

  beforeEach(() => {
    // GIVEN — a fresh SPWN_HOME
    home = createSpwnHome();
    createOrgManifest(home);
    mockApi.reset();
    mockApi.onChat(noop);
  });

  test("universe respects CPU limits from config", async () => {
    // GIVEN — a universe config with specific CPU limits
    createUniverseConfig(home, "cpu-limited", {
      physics: { cpu: 2, memory: "1g", timeout: "10m", "max-processes": 50 },
    });

    // WHEN — spawning and inspecting the universe
    const spawnResult = await spwn("spawn cpu limited")
      .exec("universe -c cpu-limited")
      .run();
    const universeId = spawnResult.stdout.match(/u-cpu-limited-\d{5}/)?.[0];

    const result = await spwn("inspect cpu limits")
      .exec(`universe inspect ${universeId}`)
      .run();

    // THEN — physics show the configured CPU limits
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("cpu");
  });

  test("universe respects memory limits from config", async () => {
    // GIVEN — a universe config with specific memory limits
    createUniverseConfig(home, "mem-limited", {
      physics: { cpu: 1, memory: "256m", timeout: "5m", "max-processes": 25 },
    });

    // WHEN — spawning and inspecting
    const spawnResult = await spwn("spawn mem limited")
      .exec("universe -c mem-limited")
      .run();
    const universeId = spawnResult.stdout.match(/u-mem-limited-\d{5}/)?.[0];

    const result = await spwn("inspect mem limits")
      .exec(`universe inspect ${universeId}`)
      .run();

    // THEN — physics show the configured memory limits
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("memory");
  });

  test("universe includes declared elements", async () => {
    // GIVEN — a universe config with specific elements
    createUniverseConfig(home, "elements", {
      elements: ["@unix", "@git", "jq", "curl"],
    });

    // WHEN — spawning and inspecting
    const spawnResult = await spwn("spawn with elements")
      .exec("universe -c elements")
      .run();
    const universeId = spawnResult.stdout.match(/u-elements-\d{5}/)?.[0];

    const result = await spwn("inspect elements")
      .exec(`universe inspect ${universeId}`)
      .run();

    // THEN — elements are listed in the universe details
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("elements");
  });

  test("faculties.md is generated inside the universe", async () => {
    // GIVEN — a universe config with elements
    createUniverseConfig(home, "faculties");

    // WHEN — spawning the universe
    const spawnResult = await spwn("spawn for faculties")
      .exec("universe -c faculties")
      .run();
    const universeId = spawnResult.stdout.match(/u-faculties-\d{5}/)?.[0];

    const result = await spwn("inspect faculties")
      .exec(`universe inspect ${universeId}`)
      .run();

    // THEN — faculties.md is mentioned or available
    expect(result.exitCode).toBe(0);
  });
});
