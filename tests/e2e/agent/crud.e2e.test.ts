import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";

describe("agent CRUD", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
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

  test("init creates agent with 6-layer Mind", async () => {
    // WHEN — initializing a new agent
    const result = await spwn("agent init")
      .exec("agent init neo")
      .run();

    // THEN — agent is created successfully
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("neo");
  });

  test("init duplicate fails", async () => {
    // GIVEN — an agent already exists
    await spwn("first init").exec("agent init neo").run();

    // WHEN — creating the same agent again
    const result = await spwn("duplicate init")
      .exec("agent init neo")
      .run();

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
  });

  test("list shows created agents", async () => {
    // GIVEN — two agents have been created
    await spwn("create neo").exec("agent init neo").run();
    await spwn("create trinity").exec("agent init trinity").run();

    // WHEN — listing agents
    const result = await spwn("list agents")
      .exec("agent list")
      .run();

    // THEN — both agents appear in the list
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("neo");
    expect(result.output).toContain("trinity");
  });

  test("inspect shows agent details", async () => {
    // GIVEN — an agent exists
    await spwn("create for inspect").exec("agent init neo").run();

    // WHEN — inspecting the agent
    const result = await spwn("inspect agent")
      .exec("agent inspect neo")
      .run();

    // THEN — details include Mind layers
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("neo");
    expect(result.output).toContain("personas");
  });

  test("list on empty home returns no agents", async () => {
    // WHEN — listing agents with no agents created
    const result = await spwn("list empty")
      .exec("agent list")
      .run();

    // THEN — exits successfully with empty or informational output
    expect(result.exitCode).toBe(0);
  });

  test("inspect non-existent agent fails", async () => {
    // WHEN — inspecting an agent that does not exist
    const result = await spwn("inspect missing")
      .exec("agent inspect nonexistent")
      .run();

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
  });
});
