import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { createAgent } from "../../setup/helpers.js";
import {
  expectLine,
  expectNoLine,
  stripAnsi,
} from "../../setup/output-helpers.js";

const COLONY_FLAGS = (home: string) => [
  "world",
  "--agent",
  "morpheus",
  "--agent",
  "neo",
  "-w",
  home,
];

describe("colony E2E", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  // ── Multi-agent spawn ──────────────────────────────────────

  test("chief and worker both appear in running world", () => {
    // GIVEN — two agents: morpheus (chief, first) and neo (worker)
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);

    // WHEN — spawning with multiple --agent flags
    const spawnResult = ctx.spwn(COLONY_FLAGS(ctx.home), 60_000);

    expect(spawnResult.exitCode).toBe(0);
    expectLine(spawnResult.output, /✓ Colony spawned\s+2 agent\(s\)/);

    // AND — world appears in ls
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();
    const listResult = ctx.spwn(["ls"]);
    expect(listResult.exitCode).toBe(0);
    const listOut = stripAnsi(listResult.output);
    expect(listOut).toContain(id);
    expect(listOut).toContain("morpheus");
    expect(listOut).toContain("neo");
  });

  test("roster.md inside container references both agents and roles", () => {
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(COLONY_FLAGS(ctx.home), 60_000);
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    const roster = ctx.world(id).readFile("/world/roster.md");
    expect(roster).toContain("morpheus");
    expect(roster).toContain("neo");
    expect(roster.toLowerCase()).toContain("chief");
    expect(roster.toLowerCase()).toContain("worker");
  });

  // ── Inter-agent messaging in colony ──────────────────────

  test("chief can send a message to worker in colony", () => {
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    ctx.spwn(COLONY_FLAGS(ctx.home), 60_000);

    const sendResult = ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "implement auth module",
    ]);
    expect(sendResult.exitCode).toBe(0);
    expectLine(sendResult.output, /Sent message\s+morpheus → neo/);
  });

  test("worker inbox shows message from chief", () => {
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    ctx.spwn(COLONY_FLAGS(ctx.home), 60_000);

    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "implement auth module",
    ]);

    const inboxResult = ctx.spwn(["agent", "inbox", "neo"]);
    expect(inboxResult.exitCode).toBe(0);
    const output = stripAnsi(inboxResult.output);
    expect(output).toContain("morpheus");
    expect(output).toContain("implement auth module");
  });

  test("message file directory exists in container after first send", () => {
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(COLONY_FLAGS(ctx.home), 60_000);
    const id = parseWorldId(spawnResult.output)!;

    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "check inbox persistence",
    ]);

    ctx.world(id).toHaveDirectory("/world/inbox/neo");
  });

  // ── Both agent dirs visible in container ──────────────────

  test("both agent home directories are mounted at /agents/<name>", () => {
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(COLONY_FLAGS(ctx.home), 60_000);
    const id = parseWorldId(spawnResult.output)!;

    ctx
      .world(id)
      .toHaveDirectory("/agents/neo")
      .toHaveDirectory("/agents/morpheus")
      .toHaveFile("/agents/neo/core/profile.md")
      .toHaveFile("/agents/morpheus/core/profile.md");
  });

  // ── Colony destroy cleanup ────────────────────────────────

  test("destroying colony world cleans up the container", () => {
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(COLONY_FLAGS(ctx.home), 60_000);
    const id = parseWorldId(spawnResult.output)!;
    ctx.world(id).toBeRunning();

    const destroyResult = ctx.spwn(["down", id], 30_000);

    expect(destroyResult.exitCode).toBe(0);
    expectLine(destroyResult.output, /✓ World destroyed\. Agent survives\./);

    ctx.world(id).toNotExist();

    const listResult = ctx.spwn(["ls"]);
    expectNoLine(
      listResult.output,
      new RegExp(id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")),
    );

    // Agents survive destruction
    ctx.mind("neo").exists();
    ctx.mind("morpheus").exists();
  });

  // ── Multiple messages in colony ───────────────────────────

  test("multiple messages between chief and worker all appear", () => {
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    ctx.spwn(COLONY_FLAGS(ctx.home), 60_000);

    ctx.spwn(["agent", "send", "neo", "--from", "morpheus", "task 1: setup database"]);
    ctx.spwn(["agent", "send", "neo", "--from", "morpheus", "task 2: implement API"]);
    ctx.spwn(["agent", "send", "neo", "--from", "morpheus", "task 3: write tests"]);

    const inboxResult = ctx.spwn(["agent", "inbox", "neo"]);
    expect(inboxResult.exitCode).toBe(0);
    const output = stripAnsi(inboxResult.output);
    expect(output).toContain("task 1: setup database");
    expect(output).toContain("task 2: implement API");
    expect(output).toContain("task 3: write tests");
  });
});
