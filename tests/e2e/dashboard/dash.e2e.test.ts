import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";
import { stripAnsi } from "../../setup/output-helpers.js";

describe("dashboard — spwn dash", () => {
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

  test("'spwn dash --help' shows subcommands", async () => {
    const result = await spwn("dash help").exec("dash --help").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("start");
    expect(out).toContain("open");
  });

  test("'spwn dash start' produces clean output even on failure", async () => {
    const result = await spwn("dash start").exec("dash start").run();

    // May fail if port is in use or server isn't configured — that's OK
    // We just verify no stack trace / unhandled crash
    const out = stripAnsi(result.output);
    expect(out).not.toContain("TypeError");
    expect(out).not.toContain("ReferenceError");
    expect(out).not.toContain("FATAL");
  });

  test("'spwn dash open' produces clean output even on failure", async () => {
    const result = await spwn("dash open").exec("dash open").run();

    // May not be able to open browser in CI — that's OK
    const out = stripAnsi(result.output);
    expect(out).not.toContain("TypeError");
    expect(out).not.toContain("ReferenceError");
    expect(out).not.toContain("FATAL");
  });
});
