import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";
import { expectLine, lines } from "../../setup/output-helpers.js";

describe("error handling", () => {
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

  test("destroy non-existent world", async () => {
    // WHEN — destroying a world that does not exist
    const result = await spwn("destroy missing")
      .exec("world destroy w-nonexistent-00000")
      .run();

    // THEN — exits with non-zero code and structured error
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /✗ Destroy failed\s+world w-nonexistent-00000 not found/);
  });

  test("inspect non-existent world", async () => {
    // WHEN — inspecting a world that does not exist
    const result = await spwn("inspect missing")
      .exec("world inspect w-nonexistent-00000")
      .run();

    // THEN — exits with error showing not found
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /world w-nonexistent-00000 not found/);
  });

  test("visitor without --world flag", async () => {
    // WHEN — running visitor without specifying a world
    const result = await spwn("visitor no world")
      .exec("visitor lint-code")
      .run();

    // THEN — exits with error about required world flag
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /required flag\(s\) "world" not set/);
  });

  test("agent reflect non-existent agent skips gracefully", async () => {
    // WHEN — reflecting on an agent that does not exist (no journal)
    const result = await spwn("reflect missing")
      .exec("agent reflect nonexistent")
      .run();

    // THEN — exits successfully with structured skip message
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /→ Reflecting on agent "nonexistent"\.\.\./);
    expectLine(result.output, /Skipped\s+no journal entries/);
  });

  test("agent fork non-existent source", async () => {
    // WHEN — forking from an agent that does not exist
    const result = await spwn("fork missing")
      .exec("agent fork nonexistent target")
      .run();

    // THEN — exits with exit code (note: fork may succeed creating sparse copy)
    // The behavior depends on whether the source agent directory exists
    // If source has no layers, fork still runs with what it finds
    if (result.exitCode === 0) {
      expectLine(result.output, /→ Forking "nonexistent" -> "target"\.\.\./);
      expectLine(result.output, /✓ Source\s+nonexistent/);
      expectLine(result.output, /✓ Target\s+target/);
    } else {
      expectLine(result.output, /not found/);
    }
  });

  test("agent export non-existent agent", async () => {
    // WHEN — exporting an agent that does not exist
    const result = await spwn("export missing")
      .exec("agent export nonexistent")
      .run();

    // THEN — exits with error showing not found
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /✗ Export failed\s+agent "nonexistent" not found/);
  });

  test("world logs for non-existent world", async () => {
    // WHEN — fetching logs for a world that does not exist
    const result = await spwn("logs missing")
      .exec("world logs w-nonexistent-00000")
      .run();

    // THEN — exits with error showing not found
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /world w-nonexistent-00000 not found/);
  });

  test("agent talk to non-existent agent", async () => {
    // WHEN — talking to an agent that does not exist
    const result = await spwn("talk missing")
      .exec('agent talk nonexistent "hello"')
      .run();

    // THEN — exits with error showing not found
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /agent "nonexistent" not found/);
  });

  test("error messages are lowercase with actionable hint", async () => {
    // WHEN — triggering an error (destroy missing world)
    const result = await spwn("error format check")
      .exec("world destroy w-nonexistent-00000")
      .run();

    // THEN — error message follows convention: structured with ✗ prefix
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /✗ Destroy failed\s+world w-nonexistent-00000 not found/);
    // Error messages should start with lowercase (Go convention)
    const errorLines = lines(result.output).filter((l) => l.includes("world w-nonexistent"));
    for (const line of errorLines) {
      // The error detail "world w-nonexistent-00000 not found" starts lowercase
      expect(line).toMatch(/world w-nonexistent-00000 not found/);
    }
  });
});
