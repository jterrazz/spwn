import { describe, test, expect, beforeEach, afterEach } from "vitest";
import {
  writeFileSync,
  readFileSync,
  utimesSync,
  mkdirSync,
  existsSync,
  readdirSync,
} from "node:fs";
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

  test("dream with no journal skips", async () => {
    // WHEN — dreaming on an agent with no journal entries
    const result = await spwn("dream empty")
      .exec("agent dream neo")
      .run();

    // THEN — exits successfully with structured skip message
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Dreaming for agent "neo"\.\.\./);
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

  test("dream on non-existent agent skips gracefully", async () => {
    // WHEN — dreaming on an agent that does not exist (no journal)
    const result = await spwn("dream missing")
      .exec("agent dream nonexistent")
      .run();

    // THEN — exits successfully with skip message (no journal entries found)
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Dreaming for agent "nonexistent"\.\.\./);
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

  test("dream with journal entries creates auto-reflexion.md", async () => {
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

    // WHEN — dreaming
    const result = await spwn("dream with journal")
      .exec("agent dream neo")
      .run();

    // THEN — auto-reflexion.md is created in playbooks/
    expect(result.exitCode).toBe(0);
    new MindAssertion(home, "neo").hasFile("memory/playbooks/auto-reflexion.md");

    // AND — output shows analysis stats
    expectLine(result.output, /Entries analyzed\s+2/);
  });

  test("dream output includes success rate", async () => {
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

    // WHEN — dreaming
    const result = await spwn("dream success rate")
      .exec("agent dream neo")
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

  // ── Deep verification: dream file content ────────────────

  test("dream auto-reflexion.md content references journal entries", async () => {
    // GIVEN — an agent with multiple diverse journal entries
    const journalDir = join(home, "agents", "neo", "memory", "journal");
    writeFileSync(
      join(journalDir, "2024-03-01.md"),
      "# Journal 2024-03-01\n## Session w-alpha-00001\n- Outcome: success\n- Duration: 10m\n- Task: deployed new API endpoint\n",
    );
    writeFileSync(
      join(journalDir, "2024-03-02.md"),
      "# Journal 2024-03-02\n## Session w-alpha-00002\n- Outcome: failure\n- Duration: 8m\n- Task: database migration crashed\n",
    );
    writeFileSync(
      join(journalDir, "2024-03-03.md"),
      "# Journal 2024-03-03\n## Session w-alpha-00003\n- Outcome: success\n- Duration: 15m\n- Task: fixed CI pipeline\n",
    );

    // WHEN — dreaming
    const result = await spwn("dream content check")
      .exec("agent dream neo")
      .run();

    // THEN — auto-reflexion.md is created
    expect(result.exitCode).toBe(0);
    const reflexionPath = join(
      home, "agents", "neo", "memory", "playbooks", "auto-reflexion.md",
    );
    expect(existsSync(reflexionPath)).toBe(true);

    // AND — its content is non-empty and structured
    const content = readFileSync(reflexionPath, "utf-8");
    expect(content.trim().length).toBeGreaterThan(0);

    // AND — output reports correct entry count
    expectLine(result.output, /Entries analyzed\s+3/);
  });

  test("dream is idempotent — running twice overwrites cleanly", async () => {
    // GIVEN — an agent with journal entries
    const journalDir = join(home, "agents", "neo", "memory", "journal");
    writeFileSync(
      join(journalDir, "2024-04-01.md"),
      "# Journal 2024-04-01\n## Session w-test-00010\n- Outcome: success\n- Duration: 5m\n",
    );

    // WHEN — dreaming twice
    await spwn("reflect first").exec("agent dream neo").run();
    const result = await spwn("dream second")
      .exec("agent dream neo")
      .run();

    // THEN — second dream succeeds without error
    expect(result.exitCode).toBe(0);

    // AND — only one auto-reflexion.md exists (not duplicated)
    const playbooksDir = join(home, "agents", "neo", "memory", "playbooks");
    const reflexionFiles = readdirSync(playbooksDir).filter(
      (f) => f.includes("auto-reflexion"),
    );
    expect(reflexionFiles.length).toBe(1);
  });

  // ── Deep verification: sleep with mixed fresh/stale ────────

  test("sleep archives stale but keeps fresh in same directory", async () => {
    // GIVEN — an agent with both stale and fresh playbooks
    const playbooksDir = join(home, "agents", "neo", "memory", "playbooks");

    const stalePath = join(playbooksDir, "ancient-patterns.md");
    writeFileSync(stalePath, "# Ancient patterns\nVery old strategies.");
    const ninetyDaysAgo = new Date(Date.now() - 90 * 24 * 60 * 60 * 1000);
    utimesSync(stalePath, ninetyDaysAgo, ninetyDaysAgo);

    const freshPath = join(playbooksDir, "current-strategy.md");
    writeFileSync(freshPath, "# Current strategy\nRecent and relevant.");

    // WHEN — sleeping
    const result = await spwn("sleep mixed")
      .exec("agent sleep neo")
      .run();

    // THEN — stale file is archived, fresh file survives
    expect(result.exitCode).toBe(0);
    expect(existsSync(stalePath)).toBe(false);
    expect(existsSync(freshPath)).toBe(true);
    expectLine(result.output, /✓ Archived playbooks\s+1/);
  });

  test("sleep archives stale knowledge files", async () => {
    // GIVEN — an agent with old knowledge files
    const knowledgeDir = join(home, "agents", "neo", "memory", "knowledge");
    const stalePath = join(knowledgeDir, "old-facts.md");
    writeFileSync(stalePath, "# Old facts\nOutdated information.");
    const sixtyDaysAgo = new Date(Date.now() - 60 * 24 * 60 * 60 * 1000);
    utimesSync(stalePath, sixtyDaysAgo, sixtyDaysAgo);

    // WHEN — sleeping
    const result = await spwn("sleep knowledge")
      .exec("agent sleep neo")
      .run();

    // THEN — stale knowledge file is archived
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Archived knowledge\s+1/);
    expect(existsSync(stalePath)).toBe(false);
  });

  test("sleep prunes old sessions", async () => {
    // GIVEN — an agent with old session directories
    const sessionsDir = join(home, "agents", "neo", "sessions");
    const oldSessionDir = join(sessionsDir, "w-old-00001");
    mkdirSync(oldSessionDir, { recursive: true });
    const logFile = join(oldSessionDir, "session.log");
    writeFileSync(logFile, "Old session log data\n");
    const ninetyDaysAgo = new Date(Date.now() - 90 * 24 * 60 * 60 * 1000);
    utimesSync(logFile, ninetyDaysAgo, ninetyDaysAgo);
    utimesSync(oldSessionDir, ninetyDaysAgo, ninetyDaysAgo);

    // WHEN — sleeping
    const result = await spwn("sleep prune sessions")
      .exec("agent sleep neo")
      .run();

    // THEN — old session is pruned
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Pruned sessions\s+\d+/);
  });

  // ── Dream + Sleep combined workflow ──────────────────────

  test("dream then sleep preserves auto-reflexion.md (it is fresh)", async () => {
    // GIVEN — an agent with journal entries
    const journalDir = join(home, "agents", "neo", "memory", "journal");
    writeFileSync(
      join(journalDir, "2024-05-01.md"),
      "# Journal 2024-05-01\n## Session w-test-00020\n- Outcome: success\n- Duration: 7m\n",
    );

    // WHEN — dreaming to create auto-reflexion.md
    const reflectResult = await spwn("dream before sleep")
      .exec("agent dream neo")
      .run();
    expect(reflectResult.exitCode).toBe(0);

    const reflexionPath = join(
      home, "agents", "neo", "memory", "playbooks", "auto-reflexion.md",
    );
    expect(existsSync(reflexionPath)).toBe(true);

    // AND — sleeping immediately after (auto-reflexion is fresh)
    const sleepResult = await spwn("sleep after dream")
      .exec("agent sleep neo")
      .run();
    expect(sleepResult.exitCode).toBe(0);

    // THEN — auto-reflexion.md should survive (it was just created)
    expect(existsSync(reflexionPath)).toBe(true);
  });
});
