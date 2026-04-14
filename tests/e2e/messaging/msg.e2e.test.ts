import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine, stripAnsi } from "../../setup/output-helpers.js";

describe("messaging - spwn agent send/inbox", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("'spwn agent send' sends a message to a running agent", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "up", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;
    expect(id).toBeTruthy();

    // WHEN - sending a message
    const result = ctx.spwn([
      "agent",
      "send",
      "neo",
      "--from",
      "morpheus",
      "hello world",
    ]);

    // THEN - message sent
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /[Ss]ent message/);
  });

  test("'spwn agent send' defaults --from to user", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    ctx.spwn(["world", "up", "--agent", "neo", "-w", ctx.home], 60_000);

    // WHEN - sending without --from
    const result = ctx.spwn(["agent", "send", "neo", "hi from default"]);

    // THEN - succeeds
    expect(result.exitCode).toBe(0);
  });

  test("'spwn agent inbox' shows messages for an agent", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    ctx.spwn(["world", "up", "--agent", "neo", "-w", ctx.home], 60_000);

    ctx.spwn([
      "agent",
      "send",
      "neo",
      "--from",
      "morpheus",
      "inbox test",
    ]);

    // WHEN - checking inbox
    const result = ctx.spwn(["agent", "inbox", "neo"]);

    // THEN - shows the message
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("morpheus");
    expect(out).toContain("inbox test");
  });

  test("'spwn agent send' to non-existent agent fails cleanly", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const result = ctx.spwn([
      "agent",
      "send",
      "nonexistent",
      "--from",
      "morpheus",
      "hello",
    ]);

    // Should fail but not crash
    expect(result.exitCode).not.toBe(0);
    const out = stripAnsi(result.output);
    expect(out).not.toContain("TypeError");
    expect(out).not.toContain("ReferenceError");
  });

  test("'spwn agent inbox' on non-existent agent fails cleanly", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const result = ctx.spwn(["agent", "inbox", "nonexistent"]);

    expect(result.exitCode).not.toBe(0);
    const out = stripAnsi(result.output);
    expect(out).not.toContain("TypeError");
    expect(out).not.toContain("ReferenceError");
  });
});
