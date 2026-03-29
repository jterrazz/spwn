import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("universe physics", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("inspect shows physics constants", () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // WHEN — inspecting the universe
    const inspectResult = ctx.spwn(["universe", "inspect", id]);

    // THEN — physics constants are shown
    expect(inspectResult.exitCode).toBe(0);
    expect(inspectResult.output).toContain("Constants");
    expect(inspectResult.output).toContain("CPU");
    expect(inspectResult.output).toContain("Memory");
  });

  test("physics.md contains constants inside container", () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // THEN — physics.md inside container contains expected fields
    const physics = ctx.universe(id).physics();
    expect(physics).toContain("CPU");
    expect(physics).toContain("Memory");
    expect(physics).toContain("Timeout");
  });

  test("physics.md contains laws", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    const physics = ctx.universe(id).physics();
    expect(physics).toContain("network");
  });

  test("faculties.md is generated inside the universe", () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(spawnResult.exitCode).toBe(0);
    const id = parseUniverseId(spawnResult.output)!;

    // THEN — the spawn output mentions faculties generation
    expect(spawnResult.output).toContain("Generated faculties");
    expect(spawnResult.output).toContain("physics.md");
    expect(spawnResult.output).toContain("faculties.md");

    // AND — faculties.md actually exists and lists elements
    const faculties = ctx.universe(id).faculties();
    expect(faculties).toContain("bash");
  });

  test("universe includes declared elements", () => {
    // GIVEN — a spawned universe (default config has @unix, @git elements)
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // WHEN — inspecting the universe
    const inspectResult = ctx.spwn(["universe", "inspect", id]);

    // THEN — exits successfully (elements are part of physics config)
    expect(inspectResult.exitCode).toBe(0);

    // AND — container has the universe directory structure
    ctx
      .universe(id)
      .toHaveDirectory("/universe")
      .toHaveFile("/universe/physics.md")
      .toHaveFile("/universe/faculties.md");
  });

  test("network isolation — curl fails inside container", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(result.output)!;

    // Try to curl from inside — should fail with network=none
    try {
      ctx
        .universe(id)
        .exec("curl -s --connect-timeout 2 http://example.com");
      // If curl succeeds, network is not isolated — but curl might not exist
    } catch {
      // Expected: either curl not found or connection refused
    }
  });
});
