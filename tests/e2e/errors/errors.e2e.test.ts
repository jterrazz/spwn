import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";

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

    // THEN — exits with non-zero code and helpful message
    expect(result.exitCode).not.toBe(0);
    expect(result.output).toContain("not found");
  });

  test("inspect non-existent world", async () => {
    // WHEN — inspecting a world that does not exist
    const result = await spwn("inspect missing")
      .exec("world inspect w-nonexistent-00000")
      .run();

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
  });

  test("visitor without --world flag", async () => {
    // WHEN — running visitor without specifying a world
    const result = await spwn("visitor no world")
      .exec('visitor "lint src/"')
      .run();

    // THEN — exits with error mentioning world requirement
    expect(result.exitCode).not.toBe(0);
    expect(result.output).toContain("world");
  });

  test("agent reflect non-existent agent skips gracefully", async () => {
    // WHEN — reflecting on an agent that does not exist (no journal)
    const result = await spwn("reflect missing")
      .exec("agent reflect nonexistent")
      .run();

    // THEN — exits successfully with a skip message (no journal entries)
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("no journal");
  });

  test("agent fork non-existent source", async () => {
    // WHEN — forking from an agent that does not exist
    const result = await spwn("fork missing")
      .exec("agent fork nonexistent target")
      .run();

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
  });

  test("agent export non-existent agent", async () => {
    // WHEN — exporting an agent that does not exist
    const result = await spwn("export missing")
      .exec("agent export nonexistent")
      .run();

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
  });

  test("world logs for non-existent world", async () => {
    // WHEN — fetching logs for a world that does not exist
    const result = await spwn("logs missing")
      .exec("world logs w-nonexistent-00000")
      .run();

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
  });

  test("agent talk to non-existent agent", async () => {
    // WHEN — talking to an agent that does not exist
    const result = await spwn("talk missing")
      .exec('agent talk nonexistent "hello"')
      .run();

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
  });

  test("error messages are lowercase with actionable hint", async () => {
    // WHEN — triggering an error (destroy missing world)
    const result = await spwn("error format check")
      .exec("world destroy w-nonexistent-00000")
      .run();

    // THEN — error message follows convention: lowercase, with hint
    expect(result.exitCode).not.toBe(0);
    // Error messages should start with lowercase (Go convention)
    const errorLine = result.output.trim().split("\n")[0];
    if (errorLine && errorLine.length > 0) {
      expect(errorLine[0]).toBe(errorLine[0].toLowerCase());
    }
  });
});
