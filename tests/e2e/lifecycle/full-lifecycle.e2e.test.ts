import { describe, test, expect, afterEach } from "vitest";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import {
  spwn,
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import {
  expectLine,
  expectNoLine,
  expectTableHeader,
  expectTableRow,
  stripAnsi,
} from "../../setup/output-helpers.js";
import { MindAssertion } from "../../setup/mind-assertion.js";

/**
 * FULL AGENT LIFECYCLE E2E TEST
 *
 * Tests the complete user journey end-to-end:
 *   init → agent new → profile → write identity → up (Docker) →
 *   ls → inspect → logs → down → journal → dream → sleep →
 *   fork → export → rm → verify cleanup
 *
 * This is the ultimate integration test — exercises the entire CLI
 * through a real user workflow with Docker containers.
 */
describe("full agent lifecycle", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("complete user journey: init → create → world → evolve → fork → cleanup", () => {
    // ── STEP 1: spwn init ──────────────────────────────────────
    ctx = createTestContext();
    const initResult = ctx.spwn(["init"]);
    expect(initResult.exitCode).toBe(0);
    expectLine(initResult.output, /✓ Created universe\s+org\.yaml/);
    expectLine(initResult.output, /✓ Created config\s+\w+\.yaml/);

    // ── STEP 2: spwn agent new neo ─────────────────────────────
    // Note: createTestContext already created "neo", but we verify
    // the agent exists via agent ls
    const lsResult1 = ctx.spwn(["agent", "ls"]);
    expect(lsResult1.exitCode).toBe(0);
    expectTableHeader(lsResult1.output, ["NAME", "LAYERS", "WORLD", "STATUS"]);
    expectTableRow(lsResult1.output, ["neo"]);

    // ── STEP 3: spwn profile neo — verify character sheet ──────
    const profileResult = ctx.spwn(["profile", "neo"]);
    expect(profileResult.exitCode).toBe(0);
    const profileOut = stripAnsi(profileResult.output);
    expect(profileOut).toContain("neo");
    expect(profileOut).toContain("Role");
    expect(profileOut).toContain("citizen");
    expect(profileOut).toContain("Identity");

    // ── STEP 4: Write identity/purpose.md manually ─────────────
    const purposePath = join(ctx.home, "agents", "neo", "identity", "purpose.md");
    writeFileSync(purposePath, "Build autonomous AI agents that evolve.\n");

    // ── STEP 5: spwn profile neo purpose — verify content ──────
    const purposeResult = ctx.spwn(["profile", "neo", "purpose"]);
    expect(purposeResult.exitCode).toBe(0);
    expect(stripAnsi(purposeResult.output)).toContain("Build autonomous AI agents");

    // ── STEP 6: spwn up --agent neo -w <workspace> (Docker) ────
    const spawnResult = ctx.spwn(
      ["up", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(spawnResult.exitCode).toBe(0);
    const worldId = parseWorldId(spawnResult.output);
    expect(worldId).toBeTruthy();
    expectLine(spawnResult.output, /✓ Created container\s+w-\w+-\d{5}/);

    // ── STEP 7: spwn ls — verify world appears ─────────────────
    const lsWorldsResult = ctx.spwn(["ls"]);
    expect(lsWorldsResult.exitCode).toBe(0);
    expect(stripAnsi(lsWorldsResult.output)).toContain(worldId!);
    expectTableHeader(lsWorldsResult.output, ["ID", "CONFIG", "AGENTS", "STATUS"]);

    // ── STEP 8: spwn world inspect <id> — verify details ───────
    const inspectWorldResult = ctx.spwn(["world", "inspect", worldId!]);
    expect(inspectWorldResult.exitCode).toBe(0);
    expectLine(
      inspectWorldResult.output,
      new RegExp(`World:\\s+${worldId!.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}`),
    );
    expectLine(inspectWorldResult.output, /Status:\s+(running|idle)/);
    expectLine(inspectWorldResult.output, /Config:\s+default/);

    // ── STEP 9: spwn world logs <id> — verify logs work ────────
    const logsResult = ctx.spwn(["world", "logs", worldId!]);
    expect(logsResult.exitCode).toBe(0);
    expect(typeof logsResult.output).toBe("string");

    // ── STEP 10: spwn down <id> — destroy world ────────────────
    const downResult = ctx.spwn(["down", worldId!], 30_000);
    expect(downResult.exitCode).toBe(0);
    expectLine(downResult.output, /✓ World destroyed\. Agent survives\./);

    // Verify world is gone from ls
    const lsAfterDown = ctx.spwn(["ls"]);
    expect(lsAfterDown.exitCode).toBe(0);
    expectNoLine(
      lsAfterDown.output,
      new RegExp(worldId!.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")),
    );

    // ── STEP 11: spwn profile neo journal — check for entries ──
    const journalResult = ctx.spwn(["profile", "neo", "journal"]);
    expect(journalResult.exitCode).toBe(0);
    // Journal may or may not have entries depending on world run time;
    // we just verify the command works
    expect(typeof journalResult.output).toBe("string");

    // ── STEP 12: spwn agent dream neo ────────────────────────
    const reflectResult = ctx.spwn(["agent", "dream", "neo"]);
    expect(reflectResult.exitCode).toBe(0);
    expectLine(reflectResult.output, /→ Dreaming for agent "neo"\.\.\./);

    // ── STEP 13: spwn agent sleep neo ──────────────────────────
    const sleepResult = ctx.spwn(["agent", "sleep", "neo"]);
    expect(sleepResult.exitCode).toBe(0);
    expectLine(sleepResult.output, /→ Sleep cycle for agent "neo"\.\.\./);

    // ── STEP 14: spwn agent fork neo neo-v2 ────────────────────
    const forkResult = ctx.spwn(["agent", "fork", "neo", "neo-v2"]);
    expect(forkResult.exitCode).toBe(0);
    expectLine(forkResult.output, /→ Forking "neo" -> "neo-v2"\.\.\./);
    expectLine(forkResult.output, /✓ Source\s+neo/);
    expectLine(forkResult.output, /✓ Target\s+neo-v2/);

    // ── STEP 15: spwn profile neo-v2 — verify fork ─────────────
    const forkProfileResult = ctx.spwn(["profile", "neo-v2"]);
    expect(forkProfileResult.exitCode).toBe(0);
    const forkProfileOut = stripAnsi(forkProfileResult.output);
    expect(forkProfileOut).toContain("neo-v2");
    expect(forkProfileOut).toContain("Role");
    expect(forkProfileOut).toContain("Identity");

    // Verify forked agent has identity layer on disk
    new MindAssertion(ctx.home, "neo-v2")
      .exists()
      .hasLayer("identity")
      .hasFile("identity/default.md");

    // ── STEP 16: spwn agent export neo ─────────────────────────
    const exportResult = ctx.spwn(["agent", "export", "neo"]);
    expect(exportResult.exitCode).toBe(0);
    expectLine(exportResult.output, /→ Exporting agent neo\.\.\./);
    expectLine(exportResult.output, /✓ Exported\s+neo\.tar\.gz/);

    // ── STEP 17: spwn agent rm neo-v2 — cleanup fork ───────────
    const rmForkResult = ctx.spwn(["agent", "rm", "neo-v2"]);
    expect(rmForkResult.exitCode).toBe(0);
    expectLine(rmForkResult.output, /✓ Deleted agent\s+neo-v2/);

    // ── STEP 18: spwn agent rm neo — cleanup original ──────────
    const rmNeoResult = ctx.spwn(["agent", "rm", "neo"]);
    expect(rmNeoResult.exitCode).toBe(0);
    expectLine(rmNeoResult.output, /✓ Deleted agent\s+neo/);

    // ── STEP 19: Verify both agents are gone ───────────────────
    const finalLsResult = ctx.spwn(["agent", "ls"]);
    expect(finalLsResult.exitCode).toBe(0);
    const finalOut = stripAnsi(finalLsResult.output);
    // Neither agent should appear in the table rows
    const neoRows = finalOut.split("\n").filter(
      (l) => /\bneo\b/.test(l) && !l.includes("NAME"),
    );
    expect(neoRows.length).toBe(0);

    // Verify directories are gone from disk
    expect(existsSync(join(ctx.home, "agents", "neo"))).toBe(false);
    expect(existsSync(join(ctx.home, "agents", "neo-v2"))).toBe(false);
  });

  test("error recovery: operations on deleted agent fail gracefully", () => {
    // GIVEN — an agent that was created then deleted
    ctx = createTestContext();
    ctx.spwn(["init"]);
    ctx.spwn(["agent", "rm", "neo"]);

    // WHEN/THEN — operations on deleted agent produce clean errors
    const inspectResult = ctx.spwn(["agent", "inspect", "neo"]);
    expect(inspectResult.exitCode).not.toBe(0);
    expectLine(inspectResult.output, /agent "neo" not found/);

    const profileResult = ctx.spwn(["profile", "neo"]);
    expect(profileResult.exitCode).not.toBe(0);
    expect(stripAnsi(profileResult.output)).toContain("not found");

    const forkResult = ctx.spwn(["agent", "fork", "neo", "neo-copy"]);
    expect(forkResult.exitCode).not.toBe(0);

    // Reflect/sleep on missing agent should still handle gracefully (no crash)
    const reflectResult = ctx.spwn(["agent", "dream", "neo"]);
    expect(reflectResult.output).not.toContain("FATAL");
    expect(reflectResult.output).not.toContain("panic");

    const sleepResult = ctx.spwn(["agent", "sleep", "neo"]);
    expect(sleepResult.output).not.toContain("FATAL");
    expect(sleepResult.output).not.toContain("panic");
  });

  test("error recovery: down on invalid world ID fails gracefully", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const result = ctx.spwn(["down", "w-fake-99999"]);
    expect(result.exitCode).not.toBe(0);
    expect(result.output).not.toContain("panic");
    expect(result.output).not.toContain("FATAL");
  });

  test("error recovery: double destroy is idempotent", () => {
    // GIVEN — a world was created and destroyed
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["up", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();
    ctx.spwn(["down", id], 30_000);

    // WHEN — destroying again
    const doubleDown = ctx.spwn(["down", id], 30_000);

    // THEN — fails gracefully without panic
    expect(doubleDown.output).not.toContain("panic");
    expect(doubleDown.output).not.toContain("FATAL");
  });
});
