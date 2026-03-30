import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("agent context (AGENT.md)", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("AGENT.md generated inside container for citizen", () => {
    // GIVEN — an initialized SPWN_HOME with agent
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning a world with a single agent
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;
    expect(id).toBeTruthy();

    // THEN — AGENT.md exists inside container
    ctx.universe(id).toHaveFile("/world/AGENT.md");

    // AND — contains citizen role
    const content = ctx.universe(id).readFile("/world/AGENT.md");
    expect(content).toContain("Citizen");
    expect(content).toContain("neo");
  });

  test("AGENT.md contains messaging instructions", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const content = ctx.universe(id).readFile("/world/AGENT.md");
    expect(content).toContain("Messaging");
    expect(content).toContain("/world/inbox");
  });

  test("AGENT.md contains world info", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const content = ctx.universe(id).readFile("/world/AGENT.md");
    expect(content).toContain("World");
    expect(content).toContain("bash");
  });

  test("AGENT.md contains Mind layer descriptions for citizen", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const content = ctx.universe(id).readFile("/world/AGENT.md");
    expect(content).toContain("/mind/personas");
    expect(content).toContain("/mind/skills");
    expect(content).toContain("/mind/knowledge");
    expect(content).toContain("/mind/journal");
  });

  test("no AGENT.md when --no-agent is used", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--no-agent", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    expect(ctx.universe(id).fileExists("/world/AGENT.md")).toBe(false);
  });
});
