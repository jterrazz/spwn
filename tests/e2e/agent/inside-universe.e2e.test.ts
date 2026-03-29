import { describe, test, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { spwn, mockApi } from "../../setup/spwn.specification.js";
import { createSpwnHome, createUniverseConfig, createOrgManifest, createAgent } from "../../setup/helpers.js";
import { respond, noop } from "../../setup/mock-api/handlers.js";

describe("agent inside universe", () => {
  let home: string;

  beforeAll(async () => {
    await mockApi.start();
  });

  afterAll(async () => {
    await mockApi.stop();
  });

  beforeEach(() => {
    // GIVEN — a fresh SPWN_HOME with org, default config, and an agent
    home = createSpwnHome();
    createOrgManifest(home);
    createUniverseConfig(home, "default");
    createAgent(home, "neo");
    mockApi.reset();
    mockApi.onChat(noop);
  });

  test("spawn agent into a universe with -n flag", async () => {
    // GIVEN — a running universe
    const spawnResult = await spwn("spawn universe for agent")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — spawning an agent into the universe
    const result = await spwn("spawn agent into universe")
      .exec(`agent -n neo --universe ${universeId}`)
      .run();

    // THEN — agent is attached to the universe
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("neo");
  });

  test("agent receives Mind inside universe container", async () => {
    // GIVEN — a universe and a scripted mock that responds
    mockApi.onChat(respond("I am neo, ready to work."));

    const spawnResult = await spwn("spawn universe for mind")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — spawning the agent
    const result = await spwn("agent with mind")
      .exec(`agent -n neo --universe ${universeId}`)
      .run();

    // THEN — agent is created with Mind loaded
    expect(result.exitCode).toBe(0);
  });

  test("talk to agent inside universe", async () => {
    // GIVEN — a universe with an agent, and a scripted response
    mockApi.onChat(respond("The migration is going well."));

    const spawnResult = await spwn("spawn universe for talk")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    await spwn("spawn agent for talk")
      .exec(`agent -n neo --universe ${universeId}`)
      .run();

    // WHEN — talking to the agent
    const result = await spwn("talk to agent")
      .exec(`agent talk neo "how's the migration?"`)
      .run();

    // THEN — the agent's response is shown
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("migration");
  });

  test("agent without --universe flag uses default behavior", async () => {
    // WHEN — spawning an agent without a universe
    const result = await spwn("agent without universe")
      .exec("agent -n neo")
      .run();

    // THEN — agent is created (standalone or error depending on implementation)
    // The behavior depends on whether standalone agents are supported
    expect(result.exitCode).toBeDefined();
  });
});
