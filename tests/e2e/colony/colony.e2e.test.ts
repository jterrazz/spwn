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

describe("colony E2E", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  // ── Governor + Citizen spawn together ─────────────────────

  test("governor and citizen both appear in running world", () => {
    // GIVEN — two agents: morpheus (governor) and neo (citizen, auto-created)
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);

    // WHEN — spawning with governor + citizen using 'up' alias
    const spawnResult = ctx.spwn(
      [
        "up",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );

    // THEN — both agents mounted
    expect(spawnResult.exitCode).toBe(0);
    expectLine(spawnResult.output, /✓ Mounted mind\s+morpheus → \/mind\/morpheus/);
    expectLine(spawnResult.output, /✓ Mounted mind\s+neo → \/mind\/neo/);
    expectLine(spawnResult.output, /✓ Colony spawned\s+2 agent\(s\)/);

    // AND — world appears in ls
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();
    const listResult = ctx.spwn(["ls"]);
    expect(listResult.exitCode).toBe(0);
    expect(stripAnsi(listResult.output)).toContain(id);
  });

  test("governor AGENT.md says tier governor", () => {
    // GIVEN — colony with governor
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // THEN — governor's AGENT.md mentions governor tier
    const agentMd = ctx.universe(id).readFile("/world/AGENT.md");
    expect(agentMd).toContain("Governor");
    expect(agentMd).toContain("morpheus");
  });

  test("citizen AGENT.md says tier citizen in colony", () => {
    // GIVEN — colony with governor + citizen
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // THEN — citizen's context references citizen role
    const agentMd = ctx.universe(id).readFile("/world/AGENT.md");
    expect(agentMd).toContain("Citizen");
    expect(agentMd).toContain("neo");
  });

  // ── Inter-agent messaging in colony ──────────────────────

  test("governor can send message to citizen in colony", () => {
    // GIVEN — colony running
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // WHEN — governor sends message to citizen
    const sendResult = ctx.spwn([
      "agent",
      "send",
      "neo",
      "--from",
      "morpheus",
      "implement auth module",
    ]);

    // THEN — message sent successfully
    expect(sendResult.exitCode).toBe(0);
    expectLine(sendResult.output, /Sent message\s+morpheus → neo/);
  });

  test("citizen inbox shows message from governor in colony", () => {
    // GIVEN — colony running with message sent
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    ctx.spwn([
      "agent",
      "send",
      "neo",
      "--from",
      "morpheus",
      "implement auth module",
    ]);

    // WHEN — checking citizen inbox
    const inboxResult = ctx.spwn(["agent", "inbox", "neo"]);

    // THEN — message appears with correct sender and content
    expect(inboxResult.exitCode).toBe(0);
    const output = stripAnsi(inboxResult.output);
    expect(output).toContain("morpheus");
    expect(output).toContain("implement auth module");
  });

  test("message file exists in container inbox directory", () => {
    // GIVEN — colony with sent message
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    ctx.spwn([
      "agent",
      "send",
      "neo",
      "--from",
      "morpheus",
      "check inbox persistence",
    ]);

    // THEN — message directory exists inside container
    ctx.universe(id).toHaveDirectory("/world/inbox/neo");
  });

  // ── Governor sees all citizens ────────────────────────────

  test("AGENT.md references both agents in colony", () => {
    // GIVEN — colony with governor + citizen
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // THEN — AGENT.md references both agent names
    const agentMd = ctx.universe(id).readFile("/world/AGENT.md");
    expect(agentMd).toContain("neo");
    expect(agentMd).toContain("morpheus");
  });

  test("both agents Mind directories exist in colony container", () => {
    // GIVEN — colony spawned
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // THEN — both agent minds are mounted
    ctx.universe(id).toHaveDirectory("/mind/neo");
    ctx.universe(id).toHaveDirectory("/mind/morpheus");
    ctx.universe(id).toHaveFile("/mind/neo/identity/default.md");
    ctx.universe(id).toHaveFile("/mind/morpheus/identity/default.md");
  });

  // ── Colony destroy cleanup ────────────────────────────────

  test("destroying colony world cleans up all agents", () => {
    // GIVEN — a running colony
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    ctx.universe(id).toBeRunning();

    // WHEN — destroying
    const destroyResult = ctx.spwn(["down", id], 30_000);

    // THEN — clean destruction
    expect(destroyResult.exitCode).toBe(0);
    expectLine(destroyResult.output, /✓ World destroyed\. Agent survives\./);

    // AND — container is gone
    ctx.universe(id).toNotExist();

    // AND — world not in list
    const listResult = ctx.spwn(["ls"]);
    expectNoLine(
      listResult.output,
      new RegExp(id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")),
    );

    // AND — agents still exist on disk (survive destruction)
    ctx.mind("neo").exists();
    ctx.mind("morpheus").exists();
  });

  // ── Colony messaging via msg alias ────────────────────────

  test("msg send alias works for colony communication", () => {
    // GIVEN — colony running
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — using msg send alias
    const sendResult = ctx.spwn([
      "msg",
      "send",
      "neo",
      "--from",
      "morpheus",
      "deploy the auth service",
    ]);

    // THEN — message sent
    expect(sendResult.exitCode).toBe(0);
    expectLine(sendResult.output, /Sent message/);
  });

  test("msg inbox alias works for colony communication", () => {
    // GIVEN — colony running with message sent
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    ctx.spwn([
      "msg",
      "send",
      "neo",
      "--from",
      "morpheus",
      "build the API",
    ]);

    // WHEN — checking inbox via alias
    const inboxResult = ctx.spwn(["msg", "inbox", "neo"]);

    // THEN — message visible
    expect(inboxResult.exitCode).toBe(0);
    const output = stripAnsi(inboxResult.output);
    expect(output).toContain("morpheus");
    expect(output).toContain("build the API");
  });

  // ── Multiple messages in colony ───────────────────────────

  test("multiple messages between governor and citizen all appear", () => {
    // GIVEN — colony running
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — sending multiple messages
    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "task 1: setup database",
    ]);
    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "task 2: implement API",
    ]);
    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "task 3: write tests",
    ]);

    // THEN — all messages appear in inbox
    const inboxResult = ctx.spwn(["agent", "inbox", "neo"]);
    expect(inboxResult.exitCode).toBe(0);
    const output = stripAnsi(inboxResult.output);
    expect(output).toContain("task 1: setup database");
    expect(output).toContain("task 2: implement API");
    expect(output).toContain("task 3: write tests");
  });
});
