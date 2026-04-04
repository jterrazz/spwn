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

describe("spwn activity CLI", () => {
  let home: string;

  beforeEach(() => {
    home = createSpwnHome();
  });

  afterEach(() => {
    spawnSync("rm", ["-rf", home], { timeout: 5000 });
  });

  test("empty log shows friendly message", () => {
    // WHEN — running activity on a fresh home
    const result = runSpwn(["activity"], home);

    // THEN — shows no activity message
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /No activity yet/i);
  });

  test("events appear after agent creation", () => {
    // GIVEN — an agent is created
    const createResult = runSpwn(["agent", "new", "neo"], home);
    expect(createResult.exitCode).toBe(0);

    // WHEN — viewing activity
    const result = runSpwn(["activity"], home);

    // THEN — agent.created event shows with natural phrase
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /agent\.created/);
    expectLine(result.output, /You created neo/);
  });

  test("events accumulate across multiple operations", () => {
    // GIVEN — create three agents
    runSpwn(["agent", "new", "neo"], home);
    runSpwn(["agent", "new", "morpheus"], home);
    runSpwn(["agent", "new", "trinity"], home);

    // WHEN — viewing activity
    const result = runSpwn(["activity"], home);

    // THEN — all three events are present
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /You created neo/);
    expectLine(result.output, /You created morpheus/);
    expectLine(result.output, /You created trinity/);
  });

  test("--limit restricts number of events", () => {
    // GIVEN — five agents created
    for (const name of ["a", "b", "c", "d", "e"]) {
      runSpwn(["agent", "new", name], home);
    }

    // WHEN — viewing with limit=2
    const result = runSpwn(["activity", "--limit", "2"], home);

    // THEN — only 2 events shown (others filtered)
    expect(result.exitCode).toBe(0);
    // Count lines with "agent.created" — should be exactly 2
    const lines = result.output.split("\n").filter((l) => l.includes("agent.created"));
    expect(lines.length).toBe(2);
  });

  test("--type filters by event type", () => {
    // GIVEN — create and delete an agent
    runSpwn(["agent", "new", "neo"], home);
    runSpwn(["agent", "rm", "neo"], home);

    // WHEN — filtering by agent.created
    const result = runSpwn(["activity", "--type", "agent.created"], home);

    // THEN — only creation event shown, no deletion
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /You created neo/);
    expectNoLine(result.output, /neo was deleted/);
  });

  test("--agent filters by agent name", () => {
    // GIVEN — two agents
    runSpwn(["agent", "new", "neo"], home);
    runSpwn(["agent", "new", "morpheus"], home);

    // WHEN — filtering by agent name
    const result = runSpwn(["activity", "--agent", "neo"], home);

    // THEN — only neo's event shown
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /You created neo/);
    expectNoLine(result.output, /morpheus/);
  });

  test("--json outputs structured events", () => {
    // GIVEN — an agent
    runSpwn(["agent", "new", "neo"], home);

    // WHEN — requesting JSON output
    const result = runSpwn(["activity", "--json"], home);

    // THEN — valid JSON with events array
    expect(result.exitCode).toBe(0);
    const parsed = JSON.parse(result.output);
    expect(parsed).toHaveProperty("events");
    expect(Array.isArray(parsed.events)).toBe(true);
    expect(parsed.events.length).toBeGreaterThan(0);

    // AND — event has expected structure
    const event = parsed.events[0];
    expect(event).toHaveProperty("id");
    expect(event).toHaveProperty("timestamp");
    expect(event).toHaveProperty("type");
    expect(event).toHaveProperty("phrase");
    expect(event).toHaveProperty("actor");
  });

  test("--json respects filters", () => {
    // GIVEN — create and delete an agent
    runSpwn(["agent", "new", "neo"], home);
    runSpwn(["agent", "rm", "neo"], home);

    // WHEN — filtering with JSON output
    const result = runSpwn(["activity", "--json", "--type", "agent.deleted"], home);

    // THEN — only deletion events in JSON
    expect(result.exitCode).toBe(0);
    const parsed = JSON.parse(result.output);
    expect(parsed.events.length).toBe(1);
    expect(parsed.events[0].type).toBe("agent.deleted");
  });

  test("output includes timestamp", () => {
    // GIVEN — an agent
    runSpwn(["agent", "new", "neo"], home);

    // WHEN — viewing activity
    const result = runSpwn(["activity"], home);

    // THEN — timestamp in HH:MM:SS format is shown
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /\d{2}:\d{2}:\d{2}/);
  });

  test("help shows command description", () => {
    const result = runSpwn(["activity", "--help"], home);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /activity log/i);
    expectLine(result.output, /--limit/);
    expectLine(result.output, /--type/);
    expectLine(result.output, /--world/);
    expectLine(result.output, /--agent/);
    expectLine(result.output, /--json/);
  });
});
