import { describe, test, expect } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";

describe("CLI output", () => {
  test("root help lists all subcommands", async () => {
    // WHEN — running spwn --help
    const result = await spwn("root help")
      .exec("--help")
      .run();

    // THEN — all top-level subcommands are listed
    expect(result.exitCode).toBe(0);
    for (const cmd of [
      "universe",
      "agent",
      "claw",
      "visitor",
      "observatory",
      "skill",
      "init",
    ]) {
      expect(result.output).toContain(cmd);
    }
  });

  test("universe help lists subcommands", async () => {
    // WHEN — running spwn universe --help
    const result = await spwn("universe help")
      .exec("universe --help")
      .run();

    // THEN — universe subcommands are listed
    expect(result.exitCode).toBe(0);
    for (const sub of ["list", "inspect", "logs", "attach", "destroy"]) {
      expect(result.output).toContain(sub);
    }
  });

  test("agent help lists subcommands", async () => {
    // WHEN — running spwn agent --help
    const result = await spwn("agent help")
      .exec("agent --help")
      .run();

    // THEN — agent subcommands are listed
    expect(result.exitCode).toBe(0);
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
      expect(result.output).toContain(sub);
    }
  });

  test("claw help lists subcommands", async () => {
    // WHEN — running spwn claw --help
    const result = await spwn("claw help")
      .exec("claw --help")
      .run();

    // THEN — claw subcommands are listed
    expect(result.exitCode).toBe(0);
    for (const sub of ["start", "stop", "status", "connect"]) {
      expect(result.output).toContain(sub);
    }
  });

  test("skill help lists subcommands", async () => {
    // WHEN — running spwn skill --help
    const result = await spwn("skill help")
      .exec("skill --help")
      .run();

    // THEN — skill subcommands are listed
    expect(result.exitCode).toBe(0);
    for (const sub of ["list", "install", "remove"]) {
      expect(result.output).toContain(sub);
    }
  });

  test("unknown command returns error", async () => {
    // WHEN — running a non-existent subcommand
    const result = await spwn("unknown cmd")
      .exec("nonexistent")
      .run();

    // THEN — exits with non-zero code
    expect(result.exitCode).not.toBe(0);
  });

  test("--json flag accepted", async () => {
    // WHEN — checking help output
    const result = await spwn("json flag")
      .exec("--help")
      .run();

    // THEN — --json is documented as a global flag
    expect(result.output).toContain("--json");
  });

  test("--quiet flag accepted", async () => {
    // WHEN — checking help output
    const result = await spwn("quiet flag")
      .exec("--help")
      .run();

    // THEN — --quiet is documented as a global flag
    expect(result.output).toContain("--quiet");
  });

  test("--verbose flag accepted", async () => {
    // WHEN — checking help output
    const result = await spwn("verbose flag")
      .exec("--help")
      .run();

    // THEN — --verbose is documented as a global flag
    expect(result.output).toContain("--verbose");
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
