import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine, stripAnsi } from "../../setup/output-helpers.js";

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
});
