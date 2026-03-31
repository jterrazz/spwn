import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { writeFileSync, utimesSync, mkdirSync, existsSync } from "node:fs";
import { join } from "node:path";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome, createAgent } from "../../setup/helpers.js";
import { expectLine } from "../../setup/output-helpers.js";
import { MindAssertion } from "../../setup/mind-assertion.js";

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
    expectLine(result.output, /✓ Layers copied\s+identity, skills, memory\/knowledge, memory\/playbooks, memory\/journal, sessions/);
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
    expectLine(inspectResult.output, /identity\/\s+default\.md/);
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

  test("reflect with journal entries creates auto-reflexion.md", async () => {
    // GIVEN — an agent with journal entries
    const journalDir = join(home, "agents", "neo", "memory", "journal");
    writeFileSync(
      join(journalDir, "2024-01-01.md"),
      "# Journal 2024-01-01\n## Session w-test-00001\n- Outcome: success\n- Duration: 5m",
    );
    writeFileSync(
      join(journalDir, "2024-01-02.md"),
      "# Journal 2024-01-02\n## Session w-test-00002\n- Outcome: failure\n- Duration: 3m",
    );

    // WHEN — reflecting
    const result = await spwn("reflect with journal")
      .exec("agent reflect neo")
      .run();

    // THEN — auto-reflexion.md is created in playbooks/
    expect(result.exitCode).toBe(0);
    new MindAssertion(home, "neo").hasFile("memory/playbooks/auto-reflexion.md");

    // AND — output shows analysis stats
    expectLine(result.output, /Entries analyzed\s+2/);
  });

  test("reflect output includes success rate", async () => {
    // GIVEN — an agent with journal entries (1 success, 1 failure)
    const journalDir = join(home, "agents", "neo", "memory", "journal");
    writeFileSync(
      join(journalDir, "2024-02-01.md"),
      "# Journal 2024-02-01\n## Session w-test-00010\n- Outcome: success\n- Duration: 2m",
    );
    writeFileSync(
      join(journalDir, "2024-02-02.md"),
      "# Journal 2024-02-02\n## Session w-test-00011\n- Outcome: failure\n- Duration: 4m",
    );

    // WHEN — reflecting
    const result = await spwn("reflect success rate")
      .exec("agent reflect neo")
      .run();

    // THEN — output contains success rate
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Success rate\s+\d+%/);
    expectLine(result.output, /Completed\s+\d+/);
    expectLine(result.output, /Failed\s+\d+/);
  });

  test("sleep archives stale playbooks", async () => {
    // GIVEN — an agent with old playbook files
    const playbooksDir = join(home, "agents", "neo", "memory", "playbooks");
    const stalePath = join(playbooksDir, "old-strategy.md");
    writeFileSync(stalePath, "# Old strategy\nThis is outdated.");
    const sixtyDaysAgo = new Date(Date.now() - 60 * 24 * 60 * 60 * 1000);
    utimesSync(stalePath, sixtyDaysAgo, sixtyDaysAgo);

    // WHEN — sleeping
    const result = await spwn("sleep stale")
      .exec("agent sleep neo")
      .run();

    // THEN — stale file is archived (removed from playbooks/)
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Archived playbooks\s+1/);
    expect(existsSync(stalePath)).toBe(false);
  });

  test("sleep preserves fresh files", async () => {
    // GIVEN — an agent with recent playbook files
    const playbooksDir = join(home, "agents", "neo", "memory", "playbooks");
    const freshPath = join(playbooksDir, "fresh-strategy.md");
    writeFileSync(freshPath, "# Fresh strategy\nThis is current.");

    // WHEN — sleeping
    const result = await spwn("sleep fresh files")
      .exec("agent sleep neo")
      .run();

    // THEN — fresh file remains in playbooks/
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Archived playbooks\s+0/);
    expect(existsSync(freshPath)).toBe(true);
  });
});
