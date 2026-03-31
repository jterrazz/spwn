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
    expect(out).toContain("Core:");
    expect(out).toContain("System:");
    // Key commands present
    for (const cmd of ["world", "agent", "init", "status", "doctor", "architect", "observatory", "skill", "upgrade", "up", "down", "ls", "profile", "msg", "snap"]) {
      expect(out).toContain(cmd);
    }
    // Flags
    expect(out).toContain("--json");
    expect(out).toContain("--quiet");
    expect(out).toContain("--verbose");
  });

  test("world help lists subcommands", async () => {
    const result = await spwn("world help").exec("world --help").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    // Custom grouped help uses section titles like "Lifecycle:", "Control:"
    for (const sub of ["list", "inspect", "logs", "attach", "destroy", "snapshot", "restore"]) {
      expect(out).toContain(sub);
    }
  });

  test("agent help lists subcommands", async () => {
    const result = await spwn("agent help").exec("agent --help").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    for (const sub of ["init", "list", "inspect", "export", "reflect", "sleep", "fork", "talk"]) {
      expect(out).toContain(sub);
    }
  });

  test("architect help lists subcommands", async () => {
    const result = await spwn("architect help").exec("architect --help").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Commands:");
    for (const sub of ["start", "stop", "status", "connect"]) {
      expect(out).toContain(sub);
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

  test("doctor runs diagnostic checks", async () => {
    const result = await spwn("doctor")
      .exec("doctor")
      .run();

    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("Docker");
    expect(stripAnsi(result.output)).toContain("Version");
  });

  test("help lists doctor command", async () => {
    const result = await spwn("help with doctor")
      .exec("--help")
      .run();

    expect(result.exitCode).toBe(0);
    expectLine(result.output, /doctor\s+/);
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
    expect(out).toContain("Anthropic");
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



  test("doctor shows Universe label", async () => {
    const result = await spwn("doctor universe")
      .exec("doctor")
      .run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Universe");
    // Should NOT contain old "Organization" label
    expect(out).not.toContain("Organization");
  });
});
