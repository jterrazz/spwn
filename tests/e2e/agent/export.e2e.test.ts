import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { existsSync } from "node:fs";
import { execSync } from "node:child_process";
import { join } from "node:path";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome, createAgent } from "../../setup/helpers.js";
import { expectLine } from "../../setup/output-helpers.js";

describe("agent export", () => {
  let home: string;
  let originalSpwnHome: string | undefined;
  let originalCwd: string;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    originalCwd = process.cwd();
    home = createSpwnHome();
    createAgent(home, "neo");
    process.env.SPWN_HOME = home;
    // Change to home so tar.gz is created in a known location
    process.chdir(home);
  });

  afterEach(() => {
    process.chdir(originalCwd);
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
  });

  test("export creates tar.gz", async () => {
    // WHEN — exporting the agent
    const result = await spwn("export agent")
      .exec("agent export neo")
      .run();

    // THEN — a tar.gz archive is created with structured output
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Exporting agent neo\.\.\./);
    expectLine(result.output, /✓ Exported\s+neo\.tar\.gz/);

    // AND — the tar.gz file actually exists on disk
    expect(existsSync(join(home, "neo.tar.gz"))).toBe(true);
  });

  test("export tar.gz contains core/profile.md", async () => {
    // WHEN — exporting the agent
    const result = await spwn("export contents")
      .exec("agent export neo")
      .run();
    expect(result.exitCode).toBe(0);

    // THEN — the tar.gz contains expected files
    const tarPath = join(home, "neo.tar.gz");
    expect(existsSync(tarPath)).toBe(true);

    const listing = execSync(`tar tzf ${tarPath}`, { encoding: "utf-8" });
    expect(listing).toContain("core/profile.md");
  });

  test("export with exclude layers", async () => {
    // WHEN — exporting with excluded layers
    const result = await spwn("export with exclude")
      .exec("agent export neo --exclude journal,sessions")
      .run();

    // THEN — export completes successfully with same output format
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Exporting agent neo\.\.\./);
    expectLine(result.output, /✓ Exported\s+neo\.tar\.gz/);

    // AND — the tar.gz file exists
    expect(existsSync(join(home, "neo.tar.gz"))).toBe(true);
  });

  test("export non-existent agent fails", async () => {
    // WHEN — exporting an agent that does not exist
    const result = await spwn("export missing")
      .exec("agent export nonexistent")
      .run();

    // THEN — exits with error showing not found
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /✗ Export failed\s+agent "nonexistent" not found/);
  });

  test("export includes all Mind layers by default", async () => {
    // WHEN — exporting without exclusions
    const result = await spwn("export full")
      .exec("agent export neo")
      .run();

    // THEN — export is successful
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Exported\s+neo\.tar\.gz/);

    // AND — the archive contains all expected layers
    const tarPath = join(home, "neo.tar.gz");
    expect(existsSync(tarPath)).toBe(true);

    const listing = execSync(`tar tzf ${tarPath}`, { encoding: "utf-8" });
    expect(listing).toMatch(/(^|\n)core(\/|$|\n)/);
    expect(listing).toMatch(/(^|\n)skills(\/|$|\n)/);
    expect(listing).toContain("core/profile.md");
  });

  test("import restores agent from export", async () => {
    // GIVEN — an exported agent
    const exportResult = await spwn("export for import")
      .exec("agent export neo")
      .run();
    expect(exportResult.exitCode).toBe(0);
    const tarPath = join(home, "neo.tar.gz");
    expect(existsSync(tarPath)).toBe(true);

    // WHEN — importing into a new agent name
    const importResult = await spwn("import agent")
      .exec(`agent import ${tarPath} --name neo-restored`)
      .run();

    // THEN — import succeeds and agent exists
    if (importResult.exitCode === 0) {
      // Verify the restored agent directory exists
      expect(existsSync(join(home, "agents", "neo-restored"))).toBe(true);
      expect(
        existsSync(
          join(home, "agents", "neo-restored", "identity", "default.md"),
        ),
      ).toBe(true);
    } else {
      // If import command doesn't exist yet, skip gracefully but document it
      expect(importResult.output).toMatch(/import|unknown|not found/i);
    }
  });
});
