import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome, createAgent } from "../../setup/helpers.js";
import { expectLine } from "../../setup/output-helpers.js";

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

    // THEN — exits successfully with structured skip message
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Reflecting on agent "neo"\.\.\./);
    expectLine(result.output, /Skipped\s+no journal entries/);
  });

  test("sleep on fresh agent — nothing to archive", async () => {
    // WHEN — putting a fresh agent to sleep
    const result = await spwn("sleep fresh")
      .exec("agent sleep neo")
      .run();

    // THEN — exits successfully with archive counts at 0
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Sleep cycle for agent "neo"\.\.\./);
    expectLine(result.output, /✓ Archived playbooks\s+0/);
    expectLine(result.output, /✓ Archived knowledge\s+0/);
    expectLine(result.output, /✓ Pruned sessions\s+0/);
  });

  test("fork creates new agent", async () => {
    // WHEN — forking an agent
    const result = await spwn("fork agent")
      .exec("agent fork neo neo-v2")
      .run();

    // THEN — new agent is created with structured output
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Forking "neo" -> "neo-v2"\.\.\./);
    expectLine(result.output, /✓ Source\s+neo/);
    expectLine(result.output, /✓ Target\s+neo-v2/);
    expectLine(result.output, /✓ Layers copied\s+personas, skills, knowledge, playbooks, journal, sessions/);
  });

  test("fork duplicate target fails", async () => {
    // GIVEN — a fork already exists
    await spwn("first fork").exec("agent fork neo neo-v2").run();

    // WHEN — forking to the same target
    const result = await spwn("duplicate fork")
      .exec("agent fork neo neo-v2")
      .run();

    // THEN — exits with error showing duplicate
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /✗ Fork failed\s+target agent "neo-v2" already exists/);
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
    expectLine(inspectResult.output, /Agent:\s+neo-clone/);
    expectLine(inspectResult.output, /personas\/\s+default\.md/);
  });

  test("reflect on non-existent agent skips gracefully", async () => {
    // WHEN — reflecting on an agent that does not exist (no journal)
    const result = await spwn("reflect missing")
      .exec("agent reflect nonexistent")
      .run();

    // THEN — exits successfully with skip message (no journal entries found)
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Reflecting on agent "nonexistent"\.\.\./);
    expectLine(result.output, /Skipped\s+no journal entries/);
  });

  test("sleep on non-existent agent is a no-op", async () => {
    // WHEN — sleeping a non-existent agent
    const result = await spwn("sleep missing")
      .exec("agent sleep nonexistent")
      .run();

    // THEN — exits successfully with archive counts at 0
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Sleep cycle for agent "nonexistent"\.\.\./);
    expectLine(result.output, /✓ Archived playbooks\s+0/);
    expectLine(result.output, /✓ Archived knowledge\s+0/);
    expectLine(result.output, /✓ Pruned sessions\s+0/);
  });
});
