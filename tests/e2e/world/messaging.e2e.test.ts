import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import {
  expectLine,
  expectTableHeader,
  stripAnsi,
} from "../../setup/output-helpers.js";

describe("agent messaging", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("send creates message in agent inbox", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const result = ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "implement webhooks",
    ]);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Sent message/);
  });

  test("inbox shows messages for an agent", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "implement webhooks",
    ]);

    const result = ctx.spwn(["agent", "inbox", "neo"]);
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("morpheus");
    expect(stripAnsi(result.output)).toContain("implement webhooks");
  });

  test("inbox shows all messages when no agent specified", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "task 1",
    ]);
    ctx.spwn([
      "agent", "send", "morpheus",
      "--from", "neo",
      "reply",
    ]);

    const result = ctx.spwn(["agent", "inbox", "neo"]);
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("morpheus");
  });

  test("inbox is empty initially", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const result = ctx.spwn(["agent", "inbox", "neo"]);
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("No messages");
  });

  test("message file exists inside container", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "check persistence",
    ]);

    ctx.universe(id).toHaveDirectory("/world/inbox/neo");
  });

  test("multiple messages to same agent all appear", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "first message",
    ]);
    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "second message",
    ]);
    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "third message",
    ]);

    const result = ctx.spwn(["agent", "inbox", "neo"]);
    expect(result.exitCode).toBe(0);
    const output = stripAnsi(result.output);
    expect(output).toContain("first message");
    expect(output).toContain("second message");
    expect(output).toContain("third message");
  });

  test("inbox shows table with correct columns", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "table test",
    ]);

    const result = ctx.spwn(["agent", "inbox", "neo"]);
    expect(result.exitCode).toBe(0);
    expectTableHeader(result.output, ["FROM", "TYPE", "STATUS"]);
  });

  test("send with --type flag sets message type", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "--type", "question",
      "what is the matrix?",
    ]);

    const result = ctx.spwn(["agent", "inbox", "neo"]);
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("question");
  });

  test("send to non-existent agent fails", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn([
      "agent", "send", "nonexistent",
      "--from", "morpheus",
      "hello",
    ]);
    expect(result.exitCode).not.toBe(0);
    expect(stripAnsi(result.output)).not.toContain("TypeError");
  });

  test("send without --from defaults to user", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    // --from is optional now; it defaults to "user".
    const result = ctx.spwn([
      "agent", "send", "neo",
      "missing from",
    ]);
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toMatch(/Sent message\s+user → neo/);
  });

  test("inbox on non-existent agent fails", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(["agent", "inbox", "nonexistent"]);
    expect(result.exitCode).not.toBe(0);
    expect(stripAnsi(result.output)).not.toContain("TypeError");
  });

  test("/world/inbox/<agent> directory exists after first send", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    // Inbox dirs are created lazily on first send.
    ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "first message — creates the inbox dir",
    ]);

    ctx.universe(id).toHaveDirectory("/world/inbox/neo");
  });

  test("physics.md includes communication section", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const physics = ctx.universe(id).physics();
    expect(physics).toContain("Communication");
    expect(physics).toContain("/world/inbox");
  });

  test("send output shows sender and recipient", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const result = ctx.spwn([
      "agent", "send", "neo",
      "--from", "morpheus",
      "output format test",
    ]);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Sent message\s+morpheus → neo/);
  });
});
