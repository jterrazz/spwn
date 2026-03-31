import { describe, test, expect } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { expectLine, lines } from "../../setup/output-helpers.js";

describe("CLI output", () => {
  test("root help lists all subcommands", async () => {
    // WHEN — running spwn --help
    const result = await spwn("root help")
      .exec("--help")
      .run();

    // THEN — custom grouped help with all sections
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Quick Start:/);
    expectLine(result.output, /World:/);
    expectLine(result.output, /Agent:/);
    expectLine(result.output, /System:/);
    expectLine(result.output, /Flags:/);
    // Key commands present (lines are trimmed, so cmd is at start)
    for (const cmd of ["world", "agent", "init", "status", "claw", "observatory", "skill"]) {
      expectLine(result.output, new RegExp(`${cmd}\\s+`));
    }
    // Flags
    expectLine(result.output, /--json/);
    expectLine(result.output, /--quiet/);
    expectLine(result.output, /--verbose/);
  });

  test("world help lists subcommands", async () => {
    // WHEN — running spwn world --help
    const result = await spwn("world help")
      .exec("world --help")
      .run();

    // THEN — world subcommands are listed
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Available Commands:/);
    for (const sub of ["list", "inspect", "logs", "attach", "destroy", "snapshot", "snapshots", "restore"]) {
      expectLine(result.output, new RegExp(`^\\s*${sub}\\s+`));
    }
  });

  test("agent help lists subcommands", async () => {
    // WHEN — running spwn agent --help
    const result = await spwn("agent help")
      .exec("agent --help")
      .run();

    // THEN — agent subcommands are listed
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Available Commands:/);
    for (const sub of [
      "init",
      "list",
      "inspect",
      "export",
      "reflect",
      "sleep",
      "fork",
      "talk",
    ]) {
      expectLine(result.output, new RegExp(`^\\s*${sub}\\s+`));
    }
  });

  test("claw help lists subcommands", async () => {
    // WHEN — running spwn claw --help
    const result = await spwn("claw help")
      .exec("claw --help")
      .run();

    // THEN — claw subcommands are listed
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Available Commands:/);
    for (const sub of ["start", "stop", "status", "connect"]) {
      expectLine(result.output, new RegExp(`^\\s*${sub}\\s+`));
    }
  });

  test("skill help lists subcommands", async () => {
    // WHEN — running spwn skill --help
    const result = await spwn("skill help")
      .exec("skill --help")
      .run();

    // THEN — skill subcommands are listed
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Available Commands:/);
    for (const sub of ["list", "install", "remove"]) {
      expectLine(result.output, new RegExp(`^\\s*${sub}\\s+`));
    }
  });

  test("unknown command returns error", async () => {
    // WHEN — running a non-existent subcommand
    const result = await spwn("unknown cmd")
      .exec("nonexistent")
      .run();

    // THEN — exits with non-zero code and helpful error
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /unknown command "nonexistent" for "spwn"/);
  });

  test("--json flag documented in help", async () => {
    // WHEN — checking help output
    const result = await spwn("json flag")
      .exec("--help")
      .run();

    // THEN — --json is documented as a global flag
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /--json\s+Output as JSON/);
  });

  test("--quiet flag documented in help", async () => {
    // WHEN — checking help output
    const result = await spwn("quiet flag")
      .exec("--help")
      .run();

    // THEN — --quiet is documented as a global flag
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /--quiet\s+Suppress/);
  });

  test("--verbose flag documented in help", async () => {
    // WHEN — checking help output
    const result = await spwn("verbose flag")
      .exec("--help")
      .run();

    // THEN — --verbose is documented as a global flag
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /--verbose\s+Debug/);
  });

  test.skip("version flag prints version", async () => {
    // TODO: spwn binary does not support --version flag yet
    const result = await spwn("version flag")
      .exec("--version")
      .run();

    expect(result.exitCode).toBe(0);
    expect(result.output).toMatch(/\d+\.\d+/);
  });
});
