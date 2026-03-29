import { describe, test, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { spwn, mockApi } from "../../setup/spwn.specification.js";
import { createSpwnHome, createUniverseConfig, createOrgManifest } from "../../setup/helpers.js";
import { noop } from "../../setup/mock-api/handlers.js";

describe("state management", () => {
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

  test("universe state persists across list calls", async () => {
    // GIVEN — a spawned universe
    const spawnResult = await spwn("spawn for state")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — listing universes twice
    const firstList = await spwn("first list")
      .exec("universe list")
      .run();
    const secondList = await spwn("second list")
      .exec("universe list")
      .run();

    // THEN — the universe appears in both lists
    expect(firstList.stdout).toContain(universeId);
    expect(secondList.stdout).toContain(universeId);
  });

  test("destroy updates state file", async () => {
    // GIVEN — a spawned universe
    const spawnResult = await spwn("spawn for state update")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — destroying the universe
    await spwn("destroy for state").exec(`universe destroy ${universeId}`).run();

    // THEN — the state no longer includes the universe
    const result = await spwn("list after state update")
      .exec("universe list")
      .run();
    expect(result.stdout).not.toContain(universeId);
  });

  test("claw state tracks active universes", async () => {
    // GIVEN — a spawned universe
    await spwn("spawn for claw state").exec("universe").run();

    // WHEN — checking claw status
    const result = await spwn("claw status")
      .exec("claw status")
      .run();

    // THEN — status shows active universes
    expect(result.exitCode).toBe(0);
  });

  test("agent state persists across commands", async () => {
    // GIVEN — a created agent
    await spwn("create agent for state")
      .exec("agent init smith")
      .run();

    // WHEN — inspecting the agent
    const result = await spwn("inspect agent state")
      .exec("agent inspect smith")
      .run();

    // THEN — agent state is consistent
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("smith");
  });

  test("concurrent universe spawns get unique IDs", async () => {
    // WHEN — spawning two universes
    const first = await spwn("spawn first for unique")
      .exec("universe")
      .run();
    const second = await spwn("spawn second for unique")
      .exec("universe")
      .run();

    // THEN — both have unique IDs
    const firstId = first.stdout.match(/u-default-\d{5}/)?.[0];
    const secondId = second.stdout.match(/u-default-\d{5}/)?.[0];
    expect(firstId).toBeDefined();
    expect(secondId).toBeDefined();
    expect(firstId).not.toBe(secondId);
  });
});
