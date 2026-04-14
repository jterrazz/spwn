import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";
import { stripAnsi, expectLine } from "../../setup/output-helpers.js";

describe("marketplace - spwn get", () => {
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

  test("'spwn get --help' shows subcommands", async () => {
    const result = await spwn("get help").exec("get --help").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("install");
    expect(out).toContain("ls");
    expect(out).toContain("search");
    expect(out).toContain("rm");
  });

  test("'spwn get ls' shows empty list when no packages installed", async () => {
    const result = await spwn("get ls").exec("get ls").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("No packages installed");
  });

  test("'spwn get install nonexistent' handles gracefully", async () => {
    const result = await spwn("get install nonexistent")
      .exec("get install nonexistent")
      .run();

    // Command should not crash - may succeed with "not yet implemented" or fail cleanly
    const out = stripAnsi(result.output);
    expect(out.length).toBeGreaterThan(0);
    // Should not contain stack traces
    expect(out).not.toContain("TypeError");
    expect(out).not.toContain("ReferenceError");
  });

  test("'spwn get rm nonexistent' handles gracefully", async () => {
    const result = await spwn("get rm nonexistent")
      .exec("get rm nonexistent")
      .run();

    // Should not crash
    const out = stripAnsi(result.output);
    expect(out.length).toBeGreaterThan(0);
    expect(out).not.toContain("TypeError");
    expect(out).not.toContain("ReferenceError");
  });

  test("'spwn get search' handles search query", async () => {
    const result = await spwn("get search test")
      .exec("get search test")
      .run();

    // Should not crash
    const out = stripAnsi(result.output);
    expect(out.length).toBeGreaterThan(0);
    expect(out).not.toContain("TypeError");
    expect(out).not.toContain("ReferenceError");
  });

  test("'spwn get' without subcommand shows help", async () => {
    const result = await spwn("get bare").exec("get").run();

    // THEN - shows help or usage info (no crash)
    const out = stripAnsi(result.output);
    expect(out.length).toBeGreaterThan(0);
    expect(out).not.toContain("TypeError");
    // Should mention available subcommands
    expect(out).toMatch(/install|search|ls|rm|help/i);
  });
});
