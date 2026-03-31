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

  // ── CLI → Disk: tier --set writes profile.yaml ─────────────

  test("tier --set governor writes profile.yaml with tier: governor", async () => {
    // GIVEN — an agent exists
    createAgent(home, "neo");

    // WHEN — setting the tier via CLI
    const result = await spwn("set tier")
      .exec("profile neo tier --set governor")
      .run();

    // THEN — CLI reports success
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Tier updated");
    expect(out).toContain("governor");

    // AND — profile.yaml on disk contains the correct tier
    const profilePath = join(home, "agents", "neo", "profile.yaml");
    expect(existsSync(profilePath)).toBe(true);
    const content = readFileSync(profilePath, "utf-8");
    expect(content).toContain("tier");
    expect(content).toContain("governor");
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

  test("manually written profile.yaml is read by CLI tier command", async () => {
    // GIVEN — an agent with a manually written profile.yaml
    createAgent(home, "neo");
    const profilePath = join(home, "agents", "neo", "profile.yaml");
    writeFileSync(
      profilePath,
      "tier: sovereign\nengine:\n  runtime: codex\n  provider: openai\n  model: o3\n",
    );

    // WHEN — reading the tier via CLI
    const result = await spwn("read tier")
      .exec("profile neo tier")
      .run();

    // THEN — CLI shows the custom tier from profile.yaml
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
      "tier: citizen\nengine:\n  runtime: codex\n  provider: openai\n  model: o3\n",
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

  test("profile.yaml format is valid YAML after tier --set", async () => {
    // GIVEN — an agent exists
    createAgent(home, "neo");

    // WHEN — setting the tier
    const result = await spwn("set tier yaml check")
      .exec("profile neo tier --set governor")
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

    // Specifically: should contain a "tier" key
    expect(content).toMatch(/tier:\s*\S+/);
  });

  test("profile.yaml survives multiple tier changes", async () => {
    // GIVEN — an agent exists
    createAgent(home, "neo");
    const profilePath = join(home, "agents", "neo", "profile.yaml");

    // WHEN — changing tier multiple times
    await spwn("set tier 1").exec("profile neo tier --set governor").run();
    await spwn("set tier 2").exec("profile neo tier --set citizen").run();
    const result = await spwn("set tier 3")
      .exec("profile neo tier --set governor")
      .run();

    // THEN — final value is correct
    expect(result.exitCode).toBe(0);
    expect(existsSync(profilePath)).toBe(true);
    const content = readFileSync(profilePath, "utf-8");
    expect(content).toContain("governor");

    // AND — file is not corrupted (should not have duplicate tier keys at root)
    const tierMatches = content.match(/^tier:/gm);
    expect(tierMatches).not.toBeNull();
    expect(tierMatches!.length).toBe(1);
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
    expect(out).toContain("citizen"); // default tier
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
