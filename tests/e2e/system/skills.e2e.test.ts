import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("system skills infrastructure", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("system AGENTS.md is injected into world containers", () => {
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

    // THEN — AGENTS.md exists inside container at /world/AGENTS.md
    ctx.universe(id).toHaveFile("/world/AGENTS.md");

    // AND — contains expected Agent Operating Manual content
    const content = ctx.universe(id).readFile("/world/AGENTS.md");
    expect(content).toBeTruthy();
    expect(content.length).toBeGreaterThan(100);
  });

  test("system skills are injected into world containers", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning a world
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;
    expect(id).toBeTruthy();

    // THEN — /world/skills/ directory exists inside container
    ctx.universe(id).toHaveDirectory("/world/skills");

    // AND — key system skill files exist
    const skillsExist = ctx.universe(id).fileExists("/world/skills");
    expect(skillsExist).toBe(true);
  });

  test("AGENT.md is generated per world with agent name", () => {
    // GIVEN — an initialized SPWN_HOME with agent
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning a world
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;
    expect(id).toBeTruthy();

    // THEN — AGENTS.md exists and references the agent name in roster
    ctx.universe(id).toHaveFile("/world/AGENTS.md");
    ctx.universe(id).toHaveFile("/world/roster.md");
    const roster = ctx.universe(id).readFile("/world/roster.md");
    expect(roster).toContain("neo");
  });

  test("agent can read system skills directory", () => {
    // GIVEN — an initialized SPWN_HOME with agent
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning a world
    const spawn = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawn.output)!;
    expect(id).toBeTruthy();

    // THEN — the skills directory is accessible
    const universe = ctx.universe(id);
    universe.toHaveDirectory("/world/skills");
  });
});
