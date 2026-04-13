import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spawnSync } from "node:child_process";
import { resolve } from "node:path";
import { createSpwnHome } from "../../setup/helpers.js";
import { expectLine, expectNoLine } from "../../setup/output-helpers.js";

const SPWN_BIN = resolve(import.meta.dirname, "../../../bin/spwn");

function runSpwn(args: string[], home: string): { exitCode: number; output: string } {
  const result = spawnSync(SPWN_BIN, args, {
    encoding: "utf-8",
    env: { ...process.env, SPWN_HOME: home, INIT_CWD: undefined } as NodeJS.ProcessEnv,
    timeout: 30_000,
  });
  return {
    exitCode: result.status ?? 1,
    output: (result.stdout ?? "") + (result.stderr ?? ""),
  };
}

describe("spwn logs CLI", () => {
  let home: string;

  beforeEach(() => {
    home = createSpwnHome();
  });

  afterEach(() => {
    spawnSync("rm", ["-rf", home], { timeout: 5000 });
  });

  test("empty log shows friendly message", () => {
    const result = runSpwn(["logs"], home);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /No events yet/i);
  });

  test("events appear after agent creation", () => {
    const createResult = runSpwn(["agent", "new", "neo"], home);
    expect(createResult.exitCode).toBe(0);

    const result = runSpwn(["logs"], home);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /agent\.created/);
    expectLine(result.output, /You created neo/);
  });

  test("events accumulate across multiple operations", () => {
    runSpwn(["agent", "new", "neo"], home);
    runSpwn(["agent", "new", "morpheus"], home);
    runSpwn(["agent", "new", "trinity"], home);

    const result = runSpwn(["logs"], home);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /You created neo/);
    expectLine(result.output, /You created morpheus/);
    expectLine(result.output, /You created trinity/);
  });

  test("--limit restricts number of events", () => {
    for (const name of ["a", "b", "c", "d", "e"]) {
      runSpwn(["agent", "new", name], home);
    }

    const result = runSpwn(["logs", "--limit", "2"], home);
    expect(result.exitCode).toBe(0);
    const lines = result.output.split("\n").filter((l) => l.includes("agent.created"));
    expect(lines.length).toBe(2);
  });

  test("--type filters by event type", () => {
    runSpwn(["agent", "new", "neo"], home);
    runSpwn(["agent", "rm", "neo"], home);

    const result = runSpwn(["logs", "--type", "agent.created"], home);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /You created neo/);
    expectNoLine(result.output, /neo was deleted/);
  });

  test("--agent filters by agent name", () => {
    runSpwn(["agent", "new", "neo"], home);
    runSpwn(["agent", "new", "morpheus"], home);

    const result = runSpwn(["logs", "--agent", "neo"], home);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /You created neo/);
    expectNoLine(result.output, /morpheus/);
  });

  test("agent logs <name> is a shortcut for --agent", () => {
    runSpwn(["agent", "new", "neo"], home);
    runSpwn(["agent", "new", "morpheus"], home);

    const result = runSpwn(["agent", "logs", "neo"], home);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /You created neo/);
    expectNoLine(result.output, /morpheus/);
  });

  test("output includes timestamp", () => {
    runSpwn(["agent", "new", "neo"], home);

    const result = runSpwn(["logs"], home);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /\d{2}:\d{2}:\d{2}/);
  });

  test("help shows command description", () => {
    const result = runSpwn(["logs", "--help"], home);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /event log/i);
    expectLine(result.output, /--limit/);
    expectLine(result.output, /--type/);
    expectLine(result.output, /--world/);
    expectLine(result.output, /--agent/);
  });
});
