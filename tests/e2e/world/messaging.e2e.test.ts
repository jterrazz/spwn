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

describe("world messaging", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("send creates message in world inbox", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const result = ctx.spwn([
      "world", "send", id,
      "--from", "morpheus",
      "--to", "neo",
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
      "world", "send", id,
      "--from", "morpheus",
      "--to", "neo",
      "implement webhooks",
    ]);

    const result = ctx.spwn(["world", "inbox", id, "neo"]);
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
      "world", "send", id,
      "--from", "morpheus",
      "--to", "neo",
      "task 1",
    ]);
    ctx.spwn([
      "world", "send", id,
      "--from", "neo",
      "--to", "morpheus",
      "reply",
    ]);

    const result = ctx.spwn(["world", "inbox", id]);
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("morpheus");
    expect(stripAnsi(result.output)).toContain("neo");
  });

  test("inbox is empty initially", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const result = ctx.spwn(["world", "inbox", id]);
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
      "world", "send", id,
      "--from", "morpheus",
      "--to", "neo",
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
      "world", "send", id,
      "--from", "morpheus",
      "--to", "neo",
      "first message",
    ]);
    ctx.spwn([
      "world", "send", id,
      "--from", "morpheus",
      "--to", "neo",
      "second message",
    ]);
    ctx.spwn([
      "world", "send", id,
      "--from", "morpheus",
      "--to", "neo",
      "third message",
    ]);

    const result = ctx.spwn(["world", "inbox", id, "neo"]);
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
      "world", "send", id,
      "--from", "morpheus",
      "--to", "neo",
      "table test",
    ]);

    const result = ctx.spwn(["world", "inbox", id]);
    expect(result.exitCode).toBe(0);
    expectTableHeader(result.output, ["FROM", "TO", "TYPE", "STATUS"]);
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
      "world", "send", id,
      "--from", "morpheus",
      "--to", "neo",
      "--type", "question",
      "what is the matrix?",
    ]);

    const result = ctx.spwn(["world", "inbox", id, "neo"]);
    expect(result.exitCode).toBe(0);
    expect(stripAnsi(result.output)).toContain("question");
  });

  test("send to non-existent world fails", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn([
      "world", "send", "w-nonexistent-00000",
      "--from", "morpheus",
      "--to", "neo",
      "hello",
    ]);
    expect(result.exitCode).not.toBe(0);
  });

  test("send without --from fails", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const result = ctx.spwn([
      "world", "send", id,
      "--to", "neo",
      "missing from",
    ]);
    expect(result.exitCode).not.toBe(0);
  });

  test("send without --to fails", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const result = ctx.spwn([
      "world", "send", id,
      "--from", "morpheus",
      "missing to",
    ]);
    expect(result.exitCode).not.toBe(0);
  });

  test("inbox on non-existent world fails", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(["world", "inbox", "w-nonexistent-00000"]);
    expect(result.exitCode).not.toBe(0);
  });

  test("/world/inbox directory exists after spawn", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    ctx.universe(id).toHaveDirectory("/world/inbox");
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
      "world", "send", id,
      "--from", "morpheus",
      "--to", "neo",
      "output format test",
    ]);
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Sent message\s+morpheus → neo/);
  });
});
