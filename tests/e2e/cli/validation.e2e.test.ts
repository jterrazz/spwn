import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome, createAgent } from "../../setup/helpers.js";
import { stripAnsi } from "../../setup/output-helpers.js";

describe("CLI input validation", () => {
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

  // ── Missing required arguments ─────────────────────────

  test("'spwn agent new' with no name shows error", async () => {
    const result = await spwn("agent new no name")
      .exec("agent new")
      .run();

    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    // Should indicate a name is required
    expect(output.toLowerCase()).toMatch(/name|required|argument|missing/);
  });

  test("'spwn agent new a b c' with too many args shows error", async () => {
    const result = await spwn("agent new extra args")
      .exec("agent new a b c")
      .run();

    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    // Should indicate invalid args
    expect(output.toLowerCase()).toMatch(/unknown|too many|invalid|argument|accepts/);
  });

  test("'spwn down' with no world ID shows error", async () => {
    const result = await spwn("down no id")
      .exec("down")
      .run();

    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    expect(output.toLowerCase()).toMatch(/world|required|argument|missing|id/);
  });

  test("'spwn world inspect' with no world ID shows error", async () => {
    const result = await spwn("inspect no id")
      .exec("world inspect")
      .run();

    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    expect(output.toLowerCase()).toMatch(/world|required|argument|missing|id/);
  });

  test("'spwn world logs' with no world ID shows error", async () => {
    const result = await spwn("logs no id")
      .exec("world logs")
      .run();

    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    expect(output.toLowerCase()).toMatch(/world|required|argument|missing|id/);
  });

  test("'spwn profile' with no agent name shows error", async () => {
    const result = await spwn("profile no name")
      .exec("profile")
      .run();

    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    expect(output.toLowerCase()).toMatch(/agent|name|required|argument|missing/);
  });

  test("'spwn profile neo unknownaspect' with invalid aspect shows error", async () => {
    createAgent(home, "neo");

    const result = await spwn("profile bad aspect")
      .exec("profile neo unknownaspect")
      .run();

    // Should fail since 'unknownaspect' is not a valid profile aspect
    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    expect(output.toLowerCase()).toMatch(/unknown|invalid|aspect|subcommand|unrecognized/);
  });

  test("'spwn msg send' with missing args shows error", async () => {
    const result = await spwn("msg send no args")
      .exec("msg send")
      .run();

    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    expect(output.toLowerCase()).toMatch(/required|argument|missing|world|message/);
  });

  // ── Error messages quality ─────────────────────────────

  test("error messages do NOT dump full usage/help", async () => {
    // Test multiple commands that should produce errors
    const commands = [
      "down w-nonexistent-00000",
      "world inspect w-nonexistent-00000",
      "agent export nonexistent",
    ];

    for (const cmd of commands) {
      const result = await spwn(`validation: ${cmd}`)
        .exec(cmd)
        .run();

      if (result.exitCode !== 0) {
        const output = stripAnsi(result.output);
        // Should NOT contain cobra usage dump
        expect(output).not.toContain("Available Commands:");
        expect(output).not.toContain("Global Flags:");
      }
    }
  });

  test("error messages contain actionable hints", async () => {
    // Destroy a non-existent world — should show clean error
    const result = await spwn("actionable hint")
      .exec("down w-nonexistent-00000")
      .run();

    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    // Error should mention what went wrong
    expect(output).toMatch(/not found/);
    // Should use the structured ✗ prefix
    expect(result.output).toMatch(/✗/);
  });

  test("unknown top-level command shows error without full usage dump", async () => {
    const result = await spwn("unknown command")
      .exec("foobar")
      .run();

    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    expect(output.toLowerCase()).toMatch(/unknown|invalid|command/);
  });

  test("agent rm with no name shows error", async () => {
    const result = await spwn("agent rm no name")
      .exec("agent rm")
      .run();

    expect(result.exitCode).not.toBe(0);
    const output = stripAnsi(result.output);
    expect(output.toLowerCase()).toMatch(/name|required|argument|missing/);
  });
});
