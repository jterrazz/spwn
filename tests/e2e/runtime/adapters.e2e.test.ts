import { describe, test, expect } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { stripAnsi } from "../../setup/output-helpers.js";

describe("runtime adapters", () => {
  test("--runtime flag appears in world help", async () => {
    const result = await spwn("world help").exec("world --help").run();
    expect(stripAnsi(result.output)).toContain("--runtime");
  });

  test("--runtime default is claude-code", async () => {
    const result = await spwn("world help").exec("world --help").run();
    expect(stripAnsi(result.output)).toContain("claude-code");
  });

  test("world runtimes subcommand exists", async () => {
    const result = await spwn("world help").exec("world --help").run();
    expect(stripAnsi(result.output)).toContain("runtimes");
  });

  const runtimes = [
    "claude-code",
    "pi",
    "codex",
    "opencode",
    "gemini",
    "aider",
  ];

  test("all runtime names appear in --runtime flag description", async () => {
    const result = await spwn("runtime flag").exec("world --help").run();
    const output = stripAnsi(result.output);
    for (const rt of runtimes) {
      expect(output).toContain(rt);
    }
  });

  test("world runtimes lists all runtimes", async () => {
    const result = await spwn("list runtimes")
      .exec("world runtimes")
      .run();
    // Should exit successfully and list runtimes
    expect(result.exitCode).toBe(0);
    const output = stripAnsi(result.output);
    for (const rt of runtimes) {
      expect(output).toContain(rt);
    }
  });
});
