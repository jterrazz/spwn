import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome, createAgent } from "../../setup/helpers.js";

describe("agent evolution", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    createAgent(home, "neo");
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
  });

  test("reflect with no journal skips", async () => {
    // WHEN — reflecting on an agent with no journal entries
    const result = await spwn("reflect empty")
      .exec("agent reflect neo")
      .run();

    // THEN — exits successfully with a skip message
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("no journal");
  });

  test("sleep on fresh agent — nothing to archive", async () => {
    // WHEN — putting a fresh agent to sleep
    const result = await spwn("sleep fresh")
      .exec("agent sleep neo")
      .run();

    // THEN — exits successfully
    expect(result.exitCode).toBe(0);
  });

  test("fork creates new agent", async () => {
    // WHEN — forking an agent
    const result = await spwn("fork agent")
      .exec("agent fork neo neo-v2")
      .run();

    // THEN — new agent is created
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("neo-v2");
  });

  test("fork duplicate target fails", async () => {
    // GIVEN — a fork already exists
    await spwn("first fork").exec("agent fork neo neo-v2").run();

    // WHEN — forking to the same target
    const result = await spwn("duplicate fork")
      .exec("agent fork neo neo-v2")
      .run();

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
  });

  test("fork preserves source Mind layers", async () => {
    // WHEN — forking an agent
    const forkResult = await spwn("fork for inspect")
      .exec("agent fork neo neo-clone")
      .run();
    expect(forkResult.exitCode).toBe(0);

    // THEN — the forked agent has the same Mind structure
    const inspectResult = await spwn("inspect forked")
      .exec("agent inspect neo-clone")
      .run();
    expect(inspectResult.exitCode).toBe(0);
    expect(inspectResult.output).toContain("personas");
  });

  test("reflect on non-existent agent skips gracefully", async () => {
    // WHEN — reflecting on an agent that does not exist (no journal)
    const result = await spwn("reflect missing")
      .exec("agent reflect nonexistent")
      .run();

    // THEN — exits successfully with skip message (no journal entries found)
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("no journal");
  });

  test("sleep on non-existent agent is a no-op", async () => {
    // WHEN — sleeping a non-existent agent
    const result = await spwn("sleep missing")
      .exec("agent sleep nonexistent")
      .run();

    // THEN — exits successfully with nothing to archive
    expect(result.exitCode).toBe(0);
  });
});
