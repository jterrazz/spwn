import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";
import { stripAnsi } from "../../setup/output-helpers.js";

describe("error recovery — state resilience", () => {
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

  test("agent commands work after failed agent operations", async () => {
    // GIVEN — an initialized home
    await spwn("init").exec("init").run();

    // WHEN — deleting a non-existent agent (trigger error)
    const errorResult = await spwn("rm ghost")
      .exec("agent rm ghost")
      .run();
    expect(errorResult.exitCode).not.toBe(0);

    // AND — creating a new agent right after
    const createResult = await spwn("create after error")
      .exec("agent new testbot")
      .run();

    // THEN — the new agent is created successfully
    expect(createResult.exitCode).toBe(0);

    // AND — it shows up in the list
    const listResult = await spwn("list after error")
      .exec("agent ls")
      .run();
    expect(listResult.exitCode).toBe(0);
    expect(stripAnsi(listResult.output)).toContain("testbot");
  });

  test("export non-existent agent does not corrupt state", async () => {
    // GIVEN — an initialized home with an agent
    await spwn("init").exec("init").run();
    await spwn("create neo").exec("agent new neo").run();

    // WHEN — exporting a non-existent agent
    const exportResult = await spwn("export ghost")
      .exec("agent export ghost")
      .run();
    expect(exportResult.exitCode).not.toBe(0);

    // THEN — the existing agent is unaffected
    const listResult = await spwn("list after bad export")
      .exec("agent ls")
      .run();
    expect(listResult.exitCode).toBe(0);
    expect(stripAnsi(listResult.output)).toContain("neo");
  });

  test("multiple errors in sequence do not compound", async () => {
    // GIVEN — an initialized home
    await spwn("init").exec("init").run();

    // WHEN — triggering multiple errors in a row
    for (let i = 0; i < 3; i++) {
      const result = await spwn(`error-${i}`)
        .exec("agent rm nonexistent")
        .run();
      expect(result.exitCode).not.toBe(0);
    }

    // THEN — a normal operation still succeeds
    const createResult = await spwn("create after errors")
      .exec("agent new survivor")
      .run();
    expect(createResult.exitCode).toBe(0);

    const listResult = await spwn("list after errors")
      .exec("agent ls")
      .run();
    expect(listResult.exitCode).toBe(0);
    expect(stripAnsi(listResult.output)).toContain("survivor");
  });

  test("init is idempotent — running init twice does not break state", async () => {
    // WHEN — running init twice
    const first = await spwn("init 1").exec("init").run();
    expect(first.exitCode).toBe(0);

    const second = await spwn("init 2").exec("init").run();
    // Second init may succeed (idempotent) or fail (already exists)
    // Either way, subsequent commands should work

    // THEN — agent commands still work
    const createResult = await spwn("create after double init")
      .exec("agent new testbot")
      .run();
    expect(createResult.exitCode).toBe(0);
  });
});
