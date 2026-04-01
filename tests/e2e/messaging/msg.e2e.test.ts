import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine, stripAnsi } from "../../setup/output-helpers.js";

describe("messaging — spwn msg aliases", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("'spwn msg --help' shows subcommands", () => {
    ctx = createTestContext();
    const result = ctx.spwn(["msg", "--help"]);

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("send");
    expect(out).toContain("inbox");
  });

  test("'spwn msg send' sends a message to agent", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;
    expect(id).toBeTruthy();

    // WHEN — sending via msg alias
    const result = ctx.spwn([
      "msg",
      "send",
      "neo",
      "--from",
      "morpheus",
      "hello from alias",
    ]);

    // THEN — message sent
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /[Ss]ent message/);
  });

  test("'spwn msg inbox' shows messages for an agent", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    ctx.spwn([
      "msg",
      "send",
      "neo",
      "--from",
      "morpheus",
      "inbox alias test",
    ]);

    // WHEN — checking inbox via msg alias
    const result = ctx.spwn(["msg", "inbox", "neo"]);

    // THEN — shows the message
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("morpheus");
    expect(out).toContain("inbox alias test");
  });

  test("'spwn msg send' to non-existent agent fails cleanly", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const result = ctx.spwn([
      "msg",
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

  test("'spwn msg inbox' on non-existent agent fails cleanly", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const result = ctx.spwn(["msg", "inbox", "nonexistent"]);

    expect(result.exitCode).not.toBe(0);
    const out = stripAnsi(result.output);
    expect(out).not.toContain("TypeError");
    expect(out).not.toContain("ReferenceError");
  });
});
