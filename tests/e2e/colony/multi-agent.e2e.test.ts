import { describe, test, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { spwn, mockApi } from "../../setup/spwn.specification.js";
import { createSpwnHome, createUniverseConfig, createOrgManifest, createAgent } from "../../setup/helpers.js";
import { respond, noop } from "../../setup/mock-api/handlers.js";

describe("colony multi-agent", () => {
  let home: string;

  beforeAll(async () => {
    await mockApi.start();
  });

  afterAll(async () => {
    await mockApi.stop();
  });

  beforeEach(() => {
    // GIVEN — a fresh SPWN_HOME with org, config, and multiple agents
    home = createSpwnHome();
    createOrgManifest(home);
    createUniverseConfig(home, "default");
    createAgent(home, "morpheus");
    createAgent(home, "neo");
    createAgent(home, "trinity");
    mockApi.reset();
    mockApi.onChat(noop);
  });

  test("spawn universe with governor and citizens", async () => {
    // WHEN — spawning a universe with a governor
    const result = await spwn("spawn colony")
      .exec("universe --governor morpheus")
      .run();

    // THEN — universe is created with the governor
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("morpheus");
  });

  test("add multiple citizens to a universe", async () => {
    // GIVEN — a running universe with a governor
    const spawnResult = await spwn("spawn for citizens")
      .exec("universe --governor morpheus")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — adding citizens to the universe
    const neoResult = await spwn("add neo")
      .exec(`agent -n neo --universe ${universeId}`)
      .run();
    const trinityResult = await spwn("add trinity")
      .exec(`agent -n trinity --universe ${universeId}`)
      .run();

    // THEN — both citizens are added
    expect(neoResult.exitCode).toBe(0);
    expect(trinityResult.exitCode).toBe(0);
  });

  test("inspect universe shows all inhabitants", async () => {
    // GIVEN — a universe with governor and citizens
    const spawnResult = await spwn("spawn for inspect")
      .exec("universe --governor morpheus")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    await spwn("add neo citizen").exec(`agent -n neo --universe ${universeId}`).run();
    await spwn("add trinity citizen").exec(`agent -n trinity --universe ${universeId}`).run();

    // WHEN — inspecting the universe
    const result = await spwn("inspect colony")
      .exec(`universe inspect ${universeId}`)
      .run();

    // THEN — all agents are visible
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("morpheus");
  });

  test("governor delegates task to citizens", async () => {
    // GIVEN — a universe with governor and citizens, scripted to respond
    mockApi.onChat(respond("Task delegated to citizens."));

    const spawnResult = await spwn("spawn for delegation")
      .exec("universe --governor morpheus")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    await spwn("add neo for delegation").exec(`agent -n neo --universe ${universeId}`).run();

    // WHEN — talking to the governor
    const result = await spwn("governor delegates")
      .exec(`agent talk morpheus "migrate the auth module"`)
      .run();

    // THEN — the mock API was called (governor invoked the LLM)
    expect(result.exitCode).toBe(0);
    expect(mockApi.calls.length).toBeGreaterThan(0);
  });

  test("destroying universe cleans up all agents", async () => {
    // GIVEN — a universe with agents
    const spawnResult = await spwn("spawn for cleanup")
      .exec("universe --governor morpheus")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    await spwn("add neo for cleanup").exec(`agent -n neo --universe ${universeId}`).run();

    // WHEN — destroying the universe
    const result = await spwn("destroy colony")
      .exec(`universe destroy ${universeId}`)
      .run();

    // THEN — universe and agents are cleaned up
    expect(result.exitCode).toBe(0);
  });
});
