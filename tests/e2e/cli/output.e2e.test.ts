import { describe, test, expect } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { expectLine, lines, stripAnsi } from "../../setup/output-helpers.js";

describe("CLI output", () => {
  test("root help lists all subcommands", async () => {
    // WHEN — running spwn --help
    const result = await spwn("root help")
      .exec("--help")
      .run();

    // THEN — custom grouped help with all sections
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Quick Start:");
    expect(out).toContain("Coordination:");
    expect(out).toContain("System:");
    // Key commands present
    for (const cmd of ["world", "agent", "init", "architect", "web", "upgrade", "up", "down", "ls", "profile"]) {
      expect(out).toContain(cmd);
    }
  });

  test("world help lists subcommands", async () => {
    const result = await spwn("world help").exec("world --help").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    // Custom grouped help uses section titles like "Lifecycle:", "Control:"
    for (const sub of ["up", "ls", "inspect", "logs", "enter", "down"]) {
      expect(out).toContain(sub);
    }
  });

  test("agent help lists subcommands", async () => {
    const result = await spwn("agent help").exec("agent --help").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    for (const sub of ["new", "ls", "show", "rm", "dream", "sleep", "fork", "talk", "send", "inbox"]) {
      expect(out).toContain(sub);
    }
  });

  test("architect help lists subcommands", async () => {
    const result = await spwn("architect help").exec("architect --help").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Commands:");
    for (const sub of ["start", "stop", "status", "talk", "logs"]) {
      expect(out).toContain(sub);
    }
  });

  test("get help lists subcommands", async () => {
    // WHEN — running spwn get --help
    const result = await spwn("get help")
      .exec("get --help")
      .run();

    // THEN — get subcommands are listed
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    for (const sub of ["install", "ls", "search", "rm"]) {
      expect(out).toContain(sub);
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

  test("--version shows version", async () => {
    const result = await spwn("version")
      .exec("--version")
      .run();

    expect(result.exitCode).toBe(0);
    expect(result.output).toMatch(/spwn version/);
  });

  test("upgrade help shows version info", async () => {
    const result = await spwn("upgrade help")
      .exec("upgrade --help")
      .run();

    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("latest");
  });

  test("help lists auth command", async () => {
    const result = await spwn("help with auth")
      .exec("--help")
      .run();

    expect(result.exitCode).toBe(0);
    expectLine(result.output, /auth\s+/);
  });

  test("auth shows authentication status", async () => {
    const result = await spwn("auth status")
      .exec("auth")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("PROVIDER");
    expect(out).toContain("STATUS");
    expect(out.toLowerCase()).toContain("anthropic");
  });

  test("auth help lists subcommands", async () => {
    const result = await spwn("auth help")
      .exec("auth --help")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("login");
    expect(out).toContain("logout");
    expect(out).toContain("token");
  });



});
