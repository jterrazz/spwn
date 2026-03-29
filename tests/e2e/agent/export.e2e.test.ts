import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome, createAgent } from "../../setup/helpers.js";
import { expectLine } from "../../setup/output-helpers.js";

describe("agent export", () => {
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

  test("export creates tar.gz", async () => {
    // WHEN — exporting the agent
    const result = await spwn("export agent")
      .exec("agent export neo")
      .run();

    // THEN — a tar.gz archive is created with structured output
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Exporting agent neo\.\.\./);
    expectLine(result.output, /✓ Exported\s+neo\.tar\.gz/);
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
  });
});
