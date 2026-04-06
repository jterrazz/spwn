import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome, createAgent } from "../../setup/helpers.js";
import { stripAnsi } from "../../setup/output-helpers.js";
import { writeFileSync, mkdirSync } from "node:fs";
import { join } from "node:path";

describe("spwn profile", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) process.env.SPWN_HOME = originalSpwnHome;
    else delete process.env.SPWN_HOME;
  });

  // ── character sheet ──────────────────────────────────────────

  test("shows character sheet for existing agent", async () => {
    createAgent(home, "neo");

    const result = await spwn("profile neo").exec("profile neo").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("neo");
    expect(out).toContain("Role");
    expect(out).toContain("worker");
    expect(out).toContain("Engine");
    expect(out).toContain("Identity");
    expect(out).toContain("Capabilities");
    expect(out).toContain("Memory");
  });

  test("shows error for nonexistent agent", async () => {
    const result = await spwn("profile ghost").exec("profile ghost").run();

    expect(result.exitCode).not.toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("not found");
    expect(out).toContain("spwn agent new");
  });

  // ── purpose ──────────────────────────────────────────────────

  test("purpose shows content when file exists", async () => {
    createAgent(home, "neo");
    writeFileSync(
      join(home, "agents", "neo", "identity", "purpose.md"),
      "Build the future of AI.\n"
    );

    const result = await spwn("profile neo purpose")
      .exec("profile neo purpose")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Build the future of AI");
  });

  test("purpose shows not-set when file missing", async () => {
    createAgent(home, "neo");

    const result = await spwn("profile neo purpose")
      .exec("profile neo purpose")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Not set yet");
  });

  // ── skills ───────────────────────────────────────────────────

  test("skills lists files when present", async () => {
    createAgent(home, "neo");
    writeFileSync(
      join(home, "agents", "neo", "skills", "deploy.md"),
      "Deploy to production servers.\n"
    );
    writeFileSync(
      join(home, "agents", "neo", "skills", "debug.md"),
      "Debug complex issues.\n"
    );

    const result = await spwn("profile neo skills")
      .exec("profile neo skills")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("deploy");
    expect(out).toContain("debug");
  });

  test("skills shows empty when no skills", async () => {
    createAgent(home, "neo");

    const result = await spwn("profile neo skills")
      .exec("profile neo skills")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("No skills yet");
  });

  // ── journal ──────────────────────────────────────────────────

  test("journal shows empty for new agent", async () => {
    createAgent(home, "neo");

    const result = await spwn("profile neo journal")
      .exec("profile neo journal")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("No sessions yet");
  });

  // ── role ─────────────────────────────────────────────────────

  test("role shows current role", async () => {
    createAgent(home, "neo");

    const result = await spwn("profile neo role")
      .exec("profile neo role")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("worker");
  });

  test("role --set chief updates profile", async () => {
    createAgent(home, "neo");

    const result = await spwn("profile neo role set")
      .exec("profile neo role --set chief")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Role updated");
    expect(out).toContain("chief");
  });

  // ── engine ───────────────────────────────────────────────────

  test("engine shows runtime info", async () => {
    createAgent(home, "neo");

    const result = await spwn("profile neo engine")
      .exec("profile neo engine")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("claude-code");
    expect(out).toContain("anthropic");
    expect(out).toContain("sonnet");
  });

  // ── sessions ─────────────────────────────────────────────────

  test("sessions shows empty for new agent", async () => {
    createAgent(home, "neo");

    const result = await spwn("profile neo sessions")
      .exec("profile neo sessions")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("No sessions yet");
  });

  // ── knowledge ────────────────────────────────────────────────

  test("knowledge shows empty for new agent", async () => {
    createAgent(home, "neo");

    const result = await spwn("profile neo knowledge")
      .exec("profile neo knowledge")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("No knowledge yet");
  });

  // ── bonds ────────────────────────────────────────────────────

  test("bonds shows not-set when file missing", async () => {
    createAgent(home, "neo");

    const result = await spwn("profile neo bonds")
      .exec("profile neo bonds")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Not set yet");
  });
});
