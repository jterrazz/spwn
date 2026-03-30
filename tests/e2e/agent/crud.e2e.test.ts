import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { existsSync } from "node:fs";
import { join } from "node:path";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";
import {
  expectLine,
  expectTableHeader,
  expectTableRow,
} from "../../setup/output-helpers.js";
import { MindAssertion } from "../../setup/mind-assertion.js";

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

    // THEN — agent is created with structured status output
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Creating agent "neo"\.\.\./);
    expectLine(result.output, /✓ Created agent\s+neo/);
    expectLine(result.output, /✓ Created persona\s+default\.md/);
    expectLine(result.output, /✓ Spawn with: spwn world --agent neo/);
  });

  test("init duplicate fails", async () => {
    // GIVEN — an agent already exists
    await spwn("first init").exec("agent init neo").run();

    // WHEN — creating the same agent again
    const result = await spwn("duplicate init")
      .exec("agent init neo")
      .run();

    // THEN — exits with error showing duplicate message
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /✗ Agent creation failed\s+agent "neo" already exists/);
  });

  test("list shows created agents", async () => {
    // GIVEN — two agents have been created
    await spwn("create neo").exec("agent init neo").run();
    await spwn("create trinity").exec("agent init trinity").run();

    // WHEN — listing agents
    const result = await spwn("list agents")
      .exec("agent list")
      .run();

    // THEN — both agents appear in a table with correct columns
    expect(result.exitCode).toBe(0);
    expectTableHeader(result.output, ["NAME", "LAYERS", "WORLD", "STATUS"]);
    expectTableRow(result.output, ["neo", "1/6", "unattached"]);
    expectTableRow(result.output, ["trinity", "1/6", "unattached"]);
  });

  test("inspect shows agent details", async () => {
    // GIVEN — an agent exists
    await spwn("create for inspect").exec("agent init neo").run();

    // WHEN — inspecting the agent
    const result = await spwn("inspect agent")
      .exec("agent inspect neo")
      .run();

    // THEN — details include agent name, path, world status, and Mind layers
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Agent:\s+neo/);
    expectLine(result.output, /World:\s+unattached/);
    expectLine(result.output, /personas\/\s+default\.md/);
    expectLine(result.output, /skills\/\s+\(empty\)/);
    expectLine(result.output, /knowledge\/\s+\(empty\)/);
    expectLine(result.output, /playbooks\/\s+\(empty\)/);
    expectLine(result.output, /journal\/\s+\(empty\)/);
    expectLine(result.output, /sessions\/\s+\(empty\)/);
  });

  test("list on empty home returns no agents", async () => {
    // WHEN — listing agents with no agents created
    const result = await spwn("list empty")
      .exec("agent list")
      .run();

    // THEN — exits successfully with table header (default agent may exist)
    expect(result.exitCode).toBe(0);
    expectTableHeader(result.output, ["NAME", "LAYERS", "WORLD", "STATUS"]);
  });

  test("inspect non-existent agent fails", async () => {
    // WHEN — inspecting an agent that does not exist
    const result = await spwn("inspect missing")
      .exec("agent inspect nonexistent")
      .run();

    // THEN — exits with error showing not found
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /agent "nonexistent" not found/);
  });

  test("delete removes agent", async () => {
    // GIVEN — an agent exists
    await spwn("create temp").exec("agent init temp").run();

    // WHEN — deleting the agent
    const result = await spwn("delete agent")
      .exec("agent delete temp")
      .run();

    // THEN — exits successfully with structured status
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Deleting agent "temp"\.\.\./);
    expectLine(result.output, /✓ Deleted agent\s+temp/);

    // AND — agent no longer appears in list
    const list = await spwn("list after delete")
      .exec("agent list")
      .run();
    const listOutput = list.output;
    // The table should not contain a row for "temp"
    const tableLines = listOutput.split("\n").filter((l) => l.includes("temp"));
    expect(tableLines.length).toBe(0);
  });

  test("delete non-existent agent fails", async () => {
    // WHEN — deleting an agent that does not exist
    const result = await spwn("delete missing")
      .exec("agent delete nonexistent")
      .run();

    // THEN — exits with error showing not found
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /✗ Delete failed\s+agent "nonexistent" not found/);
  });

  test("talk requires running world", async () => {
    // GIVEN — an agent exists but is not in any world
    await spwn("create neo for talk").exec("agent init neo").run();

    // WHEN — trying to talk to the agent
    const result = await spwn("talk without world")
      .exec("agent talk neo hello")
      .run();

    // THEN — exits with error about no active world
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /agent "neo" is not in any active world/);
    expectLine(result.output, /Spawn it first with: spwn world --agent neo/);
  });

  test("list shows world column headers", async () => {
    // GIVEN — an agent has been created
    await spwn("create for list").exec("agent init atlas").run();

    // WHEN — listing agents
    const result = await spwn("list with world")
      .exec("agent list")
      .run();

    // THEN — output includes table with world-related columns
    expect(result.exitCode).toBe(0);
    expectTableHeader(result.output, ["NAME", "LAYERS", "WORLD", "STATUS"]);
    expectTableRow(result.output, ["atlas", "1/6", "unattached"]);
  });

  test("delete actually removes Mind directory from disk", async () => {
    // GIVEN — agent exists
    await spwn("create temp for disk check").exec("agent init temp").run();
    // Verify Mind directory exists
    new MindAssertion(home, "temp").exists().hasLayer("personas");

    // WHEN — deleting the agent
    const result = await spwn("delete temp disk")
      .exec("agent delete temp")
      .run();

    // THEN — Mind directory is gone from filesystem
    expect(result.exitCode).toBe(0);
    const agentDir = join(home, "agents", "temp");
    expect(existsSync(agentDir)).toBe(false);
  });

  test("cannot inspect agent after delete", async () => {
    // GIVEN — agent is created then deleted
    await spwn("create for inspect-delete").exec("agent init temp").run();
    await spwn("delete for inspect-delete").exec("agent delete temp").run();

    // WHEN — inspecting the deleted agent
    const result = await spwn("inspect after delete")
      .exec("agent inspect temp")
      .run();

    // THEN — exits with error showing not found
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /agent "temp" not found/);
  });
});
