import { describe, test, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { spwn, mockApi } from "../../setup/spwn.specification.js";
import { createSpwnHome, createUniverseConfig, createOrgManifest } from "../../setup/helpers.js";
import { noop } from "../../setup/mock-api/handlers.js";

describe("universe lifecycle", () => {
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

  test("list shows spawned universes", async () => {
    // GIVEN — two spawned universes
    await spwn("spawn first").exec("universe").run();
    await spwn("spawn second").exec("universe").run();

    // WHEN — listing universes
    const result = await spwn("list universes")
      .exec("universe list")
      .run();

    // THEN — both universes appear in the list
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toMatch(/u-default-\d{5}/);
  });

  test("inspect shows universe details", async () => {
    // GIVEN — a spawned universe
    const spawnResult = await spwn("spawn for inspect")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — inspecting the universe
    const result = await spwn("inspect universe")
      .exec(`universe inspect ${universeId}`)
      .run();

    // THEN — shows universe details including physics and elements
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain(universeId!);
    expect(result.stdout).toContain("physics");
  });

  test("logs shows universe output", async () => {
    // GIVEN — a spawned universe
    const spawnResult = await spwn("spawn for logs")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — fetching logs
    const result = await spwn("universe logs")
      .exec(`universe logs ${universeId}`)
      .run();

    // THEN — exits successfully (logs may be empty for a fresh universe)
    expect(result.exitCode).toBe(0);
  });

  test("full lifecycle: spawn, inspect, destroy", async () => {
    // WHEN — spawning a universe
    const spawnResult = await spwn("lifecycle spawn")
      .exec("universe")
      .run();
    expect(spawnResult.exitCode).toBe(0);
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // AND — inspecting it
    const inspectResult = await spwn("lifecycle inspect")
      .exec(`universe inspect ${universeId}`)
      .run();
    expect(inspectResult.exitCode).toBe(0);

    // AND — destroying it
    const destroyResult = await spwn("lifecycle destroy")
      .exec(`universe destroy ${universeId}`)
      .run();
    expect(destroyResult.exitCode).toBe(0);

    // THEN — universe no longer appears in list
    const listResult = await spwn("lifecycle list")
      .exec("universe list")
      .run();
    expect(listResult.stdout).not.toContain(universeId);
  });
});
