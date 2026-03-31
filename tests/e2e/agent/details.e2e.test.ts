import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome, createAgent } from "../../setup/helpers.js";
import { expectLine, expectTableHeader, stripAnsi } from "../../setup/output-helpers.js";

describe("agent detail commands", () => {
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

  // ── mind command ──────────────────────────────────────────

  test("mind shows Mind directory tree", async () => {
    // GIVEN — an agent exists
    await spwn("create").exec("agent init neo").run();

    // WHEN — showing mind
    const result = await spwn("mind").exec("agent mind neo").run();

    // THEN — shows Mind structure
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("neo");
    expect(stripAnsi(result.output)).toContain("identity");
    expect(stripAnsi(result.output)).toContain("default.md");
  });

  test("mind shows empty layers", async () => {
    await spwn("create").exec("agent init neo").run();
    const result = await spwn("mind").exec("agent mind neo").run();

    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("(empty)");
  });

  test("mind on non-existent agent fails", async () => {
    const result = await spwn("mind missing").exec("agent mind ghost").run();
    expect(result.exitCode).not.toBe(0);
  });

  // ── stats command ─────────────────────────────────────────

  test("stats shows agent overview", async () => {
    await spwn("create").exec("agent init neo").run();
    const result = await spwn("stats").exec("agent stats neo").run();

    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("neo");
    expect(stripAnsi(result.output)).toContain("Sessions");
    expect(stripAnsi(result.output)).toContain("LAYER");
  });

  test("stats on non-existent agent fails", async () => {
    const result = await spwn("stats missing").exec("agent stats ghost").run();
    expect(result.exitCode).not.toBe(0);
  });

  // ── journal command ───────────────────────────────────────

  test("journal on fresh agent shows no entries", async () => {
    await spwn("create").exec("agent init neo").run();
    const result = await spwn("journal").exec("agent journal neo").run();

    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("No journal");
  });

  test("journal on non-existent agent fails", async () => {
    const result = await spwn("journal missing").exec("agent journal ghost").run();
    expect(result.exitCode).not.toBe(0);
  });

  // ── sessions command ──────────────────────────────────────

  test("sessions on fresh agent shows no sessions", async () => {
    await spwn("create").exec("agent init neo").run();
    const result = await spwn("sessions").exec("agent sessions neo").run();

    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("No sessions");
  });

  test("sessions on non-existent agent fails", async () => {
    const result = await spwn("sessions missing").exec("agent sessions ghost").run();
    expect(result.exitCode).not.toBe(0);
  });

  // ── doctor command ────────────────────────────────────────

  test("doctor runs all checks", async () => {
    await spwn("init").exec("init").run();
    const result = await spwn("doctor").exec("doctor").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Docker");
    expect(out).toContain("Config");
    expect(out).toContain("Version");
    expect(out).toContain("checks passed");
  });

  // ── upgrade command ───────────────────────────────────────

  test("upgrade help shows description", async () => {
    const result = await spwn("upgrade help").exec("upgrade --help").run();
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("latest");
    expect(stripAnsi(result.output)).toContain("GitHub");
  });
});
