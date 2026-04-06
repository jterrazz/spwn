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

  test("AGENT.md generated inside container for worker", () => {
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

    // AND — contains worker role
    const content = ctx.universe(id).readFile("/world/AGENT.md");
    expect(content).toContain("Worker");
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

  test("AGENT.md contains Mind layer descriptions for worker (new structure)", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    const content = ctx.universe(id).readFile("/world/AGENT.md");
    // New directory structure: identity, skills, memory/knowledge, memory/playbooks, memory/journal
    expect(content).toContain("/mind/identity/");
    expect(content).toContain("/mind/skills/");
    expect(content).toContain("/mind/memory/knowledge/");
    expect(content).toContain("/mind/memory/playbooks/");
    expect(content).toContain("/mind/memory/journal/");
    // Should NOT contain old paths
    expect(content).not.toContain("/mind/personas");
    expect(content).not.toContain("/mind/knowledge/");
  });

  test("container has new Mind directory structure mounted", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;

    // Verify the new directory structure inside the container
    ctx.universe(id)
      .toHaveDirectory("/mind/identity")
      .toHaveDirectory("/mind/skills")
      .toHaveDirectory("/mind/memory/knowledge")
      .toHaveDirectory("/mind/memory/playbooks")
      .toHaveDirectory("/mind/memory/journal")
      .toHaveDirectory("/mind/sessions")
      .toHaveFile("/mind/identity/default.md");
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
