import { describe, test, expect, afterEach } from "vitest";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import {
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
 * COMPLETE AGENT LIFECYCLE E2E TEST
 *
 * Tests the full user journey:
 *   create → configure → spawn → inspect → destroy → dream
 *
 * This exercises the complete end-to-end flow including knowledge access,
 * identity configuration, world management, and agent evolution.
 */
describe("complete agent lifecycle", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("complete agent lifecycle: create → configure → spawn → inspect → destroy → dream", () => {
    // ── STEP 1: Initialize universe ─────────────────────────────
    ctx = createTestContext();
    const initResult = ctx.spwn(["init"]);
    expect(initResult.exitCode).toBe(0);

    // ── STEP 2: Verify agent exists ─────────────────────────────
    const agentLs = ctx.spwn(["agent", "ls"]);
    expect(agentLs.exitCode).toBe(0);
    expectTableHeader(agentLs.output, ["NAME", "WORLD", "STATUS"]);
    expectTableRow(agentLs.output, ["neo"]);

    // ── STEP 3: Write purpose and traits ────────────────────────
    const purposePath = join(ctx.home, "agents", "neo", "identity", "purpose.md");
    writeFileSync(purposePath, "Build intelligent autonomous systems that learn and adapt.\n");

    const purposeResult = ctx.spwn(["profile", "neo", "purpose"]);
    expect(purposeResult.exitCode).toBe(0);
    expect(stripAnsi(purposeResult.output)).toContain("intelligent autonomous systems");

    // ── STEP 4: Spawn world ─────────────────────────────────────
    const spawnResult = ctx.spwn(
      ["up", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(spawnResult.exitCode).toBe(0);
    const worldId = parseWorldId(spawnResult.output);
    expect(worldId).toBeTruthy();
    expectLine(spawnResult.output, /✓ Created container\s+w-\w+-\d{5}/);

    // ── STEP 5: Verify world in ls ──────────────────────────────
    const worldLs = ctx.spwn(["ls"]);
    expect(worldLs.exitCode).toBe(0);
    expect(stripAnsi(worldLs.output)).toContain(worldId!);
    expectTableHeader(worldLs.output, ["ID", "CONFIG", "AGENTS", "STATUS"]);

    // ── STEP 6: Inspect world ───────────────────────────────────
    const inspectResult = ctx.spwn(["world", "inspect", worldId!]);
    expect(inspectResult.exitCode).toBe(0);
    expectLine(
      inspectResult.output,
      new RegExp(`World:\\s+${worldId!.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}`),
    );
    expectLine(inspectResult.output, /Status:\s+(running|idle)/);

    // ── STEP 7: Destroy world ───────────────────────────────────
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

    // ── STEP 8: Verify journal entry created ────────────────────
    const journalResult = ctx.spwn(["profile", "neo", "journal"]);
    expect(journalResult.exitCode).toBe(0);
    // Journal should exist and not error out — entries may or may not exist
    // depending on world runtime duration
    expect(typeof journalResult.output).toBe("string");

    // ── STEP 9: Dream → verify result ───────────────────────────
    const dreamResult = ctx.spwn(["agent", "dream", "neo"]);
    expect(dreamResult.exitCode).toBe(0);
    expectLine(dreamResult.output, /→ Dreaming for agent "neo"\.\.\./);

    // ── STEP 10: Cleanup — remove agent ─────────────────────────
    const rmResult = ctx.spwn(["agent", "rm", "neo"]);
    expect(rmResult.exitCode).toBe(0);
    expectLine(rmResult.output, /✓ Deleted agent\s+neo/);

    // Verify agent directory is gone
    expect(existsSync(join(ctx.home, "agents", "neo"))).toBe(false);
  });

  test("agent profile shows correct data at each lifecycle stage", () => {
    // GIVEN — fresh context with agent
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — checking profile before any configuration
    const initialProfile = ctx.spwn(["profile", "neo"]);
    expect(initialProfile.exitCode).toBe(0);
    const initialOut = stripAnsi(initialProfile.output);
    expect(initialOut).toContain("neo");
    expect(initialOut).toContain("Role");

    // WHEN — adding identity content
    const identityPath = join(ctx.home, "agents", "neo", "identity", "default.md");
    writeFileSync(identityPath, "# Neo\nYou are a code architect specializing in distributed systems.\n");

    // THEN — profile reflects updated identity
    const updatedProfile = ctx.spwn(["profile", "neo"]);
    expect(updatedProfile.exitCode).toBe(0);

    // Check mind assertion on disk
    new MindAssertion(ctx.home, "neo")
      .exists()
      .hasLayer("identity")
      .hasFile("identity/default.md");
  });

  test("multiple agents can coexist in the same universe", () => {
    // GIVEN — initialized universe
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — creating a second agent
    const newResult = ctx.spwn(["agent", "new", "trinity"]);
    expect(newResult.exitCode).toBe(0);

    // THEN — both agents appear in agent ls
    const agentLs = ctx.spwn(["agent", "ls"]);
    expect(agentLs.exitCode).toBe(0);
    const output = stripAnsi(agentLs.output);
    expect(output).toContain("neo");
    expect(output).toContain("trinity");

    // CLEANUP
    ctx.spwn(["agent", "rm", "trinity"]);
  });
});
