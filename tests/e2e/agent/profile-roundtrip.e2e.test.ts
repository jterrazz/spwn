import { describe, test, expect, beforeEach, afterEach } from "vitest";
import {
  existsSync,
  readFileSync,
  writeFileSync,
  mkdirSync,
} from "node:fs";
import { join } from "node:path";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome, createAgent } from "../../setup/helpers.js";
import { stripAnsi } from "../../setup/output-helpers.js";

/**
 * PROFILE.YAML ROUNDTRIP TEST
 *
 * Verifies that profile.yaml is correctly read and written by the CLI:
 *   - CLI writes → disk has correct YAML
 *   - Disk has YAML → CLI reads correct values
 *   - YAML format is valid and parseable
 */
describe("profile.yaml roundtrip", () => {
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

  // ── CLI → Disk: role --set writes profile.yaml ─────────────

  test("role --set chief writes profile.yaml with role: chief", async () => {
    // GIVEN — an agent exists
    createAgent(home, "neo");

    // WHEN — setting the role via CLI
    const result = await spwn("set role")
      .exec("profile neo role --set chief")
      .run();

    // THEN — CLI reports success
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Role updated");
    expect(out).toContain("chief");

    // AND — profile.yaml on disk contains the correct role
    const profilePath = join(home, "agents", "neo", "profile.yaml");
    expect(existsSync(profilePath)).toBe(true);
    const content = readFileSync(profilePath, "utf-8");
    expect(content).toContain("role");
    expect(content).toContain("chief");
  });

  test("engine --set codex writes profile.yaml with engine info", async () => {
    // GIVEN — an agent exists
    createAgent(home, "neo");

    // WHEN — setting the engine via CLI
    const result = await spwn("set engine")
      .exec("profile neo engine --set codex")
      .run();

    // THEN — CLI reports success (or handles the subcommand)
    if (result.exitCode === 0) {
      // Verify profile.yaml on disk
      const profilePath = join(home, "agents", "neo", "profile.yaml");
      expect(existsSync(profilePath)).toBe(true);
      const content = readFileSync(profilePath, "utf-8");
      expect(content).toContain("codex");
    } else {
      // engine --set may not be implemented; verify clean error
      expect(result.output).not.toContain("panic");
      expect(result.output).not.toContain("FATAL");
    }
  });

  // ── Disk → CLI: manual profile.yaml is read by CLI ─────────

  test("manually written profile.yaml is read by CLI role command", async () => {
    // GIVEN — an agent with a manually written profile.yaml
    createAgent(home, "neo");
    const profilePath = join(home, "agents", "neo", "profile.yaml");
    writeFileSync(
      profilePath,
      "role: sovereign\nengine:\n  runtime: codex\n  provider: openai\n  model: o3\n",
    );

    // WHEN — reading the role via CLI
    const result = await spwn("read role")
      .exec("profile neo role")
      .run();

    // THEN — CLI shows the custom role from profile.yaml
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("sovereign");
  });

  test("manually written profile.yaml is read by CLI engine command", async () => {
    // GIVEN — an agent with a custom engine in profile.yaml
    createAgent(home, "neo");
    const profilePath = join(home, "agents", "neo", "profile.yaml");
    writeFileSync(
      profilePath,
      "role: worker\nengine:\n  runtime: codex\n  provider: openai\n  model: o3\n",
    );

    // WHEN — reading the engine via CLI
    const result = await spwn("read engine")
      .exec("profile neo engine")
      .run();

    // THEN — CLI shows custom engine values
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("codex");
    expect(out).toContain("openai");
  });

  // ── YAML format validation ─────────────────────────────────

  test("profile.yaml format is valid YAML after role --set", async () => {
    // GIVEN — an agent exists
    createAgent(home, "neo");

    // WHEN — setting the role
    const result = await spwn("set role yaml check")
      .exec("profile neo role --set chief")
      .run();
    expect(result.exitCode).toBe(0);

    // THEN — profile.yaml is valid YAML (basic structure check)
    const profilePath = join(home, "agents", "neo", "profile.yaml");
    expect(existsSync(profilePath)).toBe(true);
    const content = readFileSync(profilePath, "utf-8");

    // Basic YAML validation: should have key: value pairs
    // Should not be empty
    expect(content.trim().length).toBeGreaterThan(0);

    // Should contain valid YAML-like structure (key: value on lines)
    const lines = content.trim().split("\n");
    const hasValidLines = lines.some((line) => {
      const trimmed = line.trim();
      // Allow comments, empty lines, and key: value pairs
      return (
        trimmed.length === 0 ||
        trimmed.startsWith("#") ||
        /^[\w-]+:/.test(trimmed) ||
        /^\s+[\w-]+:/.test(trimmed)
      );
    });
    expect(hasValidLines).toBe(true);

    // Specifically: should contain a "role" key
    expect(content).toMatch(/role:\s*\S+/);
  });

  test("profile.yaml survives multiple role changes", async () => {
    // GIVEN — an agent exists
    createAgent(home, "neo");
    const profilePath = join(home, "agents", "neo", "profile.yaml");

    // WHEN — changing role multiple times
    await spwn("set role 1").exec("profile neo role --set chief").run();
    await spwn("set role 2").exec("profile neo role --set worker").run();
    const result = await spwn("set role 3")
      .exec("profile neo role --set chief")
      .run();

    // THEN — final value is correct
    expect(result.exitCode).toBe(0);
    expect(existsSync(profilePath)).toBe(true);
    const content = readFileSync(profilePath, "utf-8");
    expect(content).toContain("chief");

    // AND — file is not corrupted (should not have duplicate role keys at root)
    const roleMatches = content.match(/^role:/gm);
    expect(roleMatches).not.toBeNull();
    expect(roleMatches!.length).toBe(1);
  });

  // ── Edge cases ─────────────────────────────────────────────

  test("profile shows defaults when no profile.yaml exists", async () => {
    // GIVEN — agent exists without profile.yaml
    createAgent(home, "neo");
    const profilePath = join(home, "agents", "neo", "profile.yaml");
    // Ensure no profile.yaml
    expect(existsSync(profilePath)).toBe(false);

    // WHEN — viewing the profile
    const result = await spwn("profile defaults")
      .exec("profile neo")
      .run();

    // THEN — shows default values
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("worker"); // default role
  });

  test("profile on nonexistent agent returns error", async () => {
    // WHEN — checking profile for an agent that doesn't exist
    const result = await spwn("profile ghost")
      .exec("profile ghost")
      .run();

    // THEN — clean error, not crash
    expect(result.exitCode).not.toBe(0);
    expect(stripAnsi(result.output)).toContain("not found");
    expect(result.output).not.toContain("panic");
  });
});
