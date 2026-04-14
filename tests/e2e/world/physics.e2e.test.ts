import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine } from "../../setup/output-helpers.js";

describe("world physics", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("inspect shows physics constants", () => {
    // GIVEN - a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN - inspecting the world
    const inspectResult = ctx.spwn(["world", "inspect", id]);

    // THEN - physics constants are shown in structured format
    expect(inspectResult.exitCode).toBe(0);
    expectLine(inspectResult.output, /Constants:\s+CPU:.*Memory:.*Timeout:/);
    expectLine(inspectResult.output, /Laws:\s+Network:/);
  });

  test("physics.md contains constants inside container", () => {
    // GIVEN - a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // THEN - physics.md inside container contains expected fields
    const physics = ctx.world(id).physics();
    expect(physics).toMatch(/CPU/);
    expect(physics).toMatch(/Memory/);
    expect(physics).toMatch(/Timeout/);
  });

  test("physics.md contains laws", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    const physics = ctx.world(id).physics();
    expect(physics).toMatch(/network/i);
  });

  test("faculties.md is generated inside the world", () => {
    // GIVEN - a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(spawnResult.exitCode).toBe(0);
    const id = parseWorldId(spawnResult.output)!;

    // THEN - faculties.md exists and lists tools
    const faculties = ctx.world(id).faculties();
    expect(faculties).toMatch(/bash/);
  });

  test("world includes declared elements", () => {
    // GIVEN - a spawned world (default config has @spwn/unix, @spwn/git elements)
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN - inspecting the world
    const inspectResult = ctx.spwn(["world", "inspect", id]);

    // THEN - exits successfully (elements are part of physics config)
    expect(inspectResult.exitCode).toBe(0);

    // AND - container has the world directory structure
    ctx
      .world(id)
      .toHaveDirectory("/world")
      .toHaveFile("/world/physics.md")
      .toHaveFile("/world/faculties.md");
  });

  test("default network mode is bridge", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(result.output)!;

    // Verify network mode is set to bridge (the current default)
    const inspectData = ctx.world(id).inspect();
    expect(inspectData.HostConfig?.NetworkMode).toBe("bridge");
  });
});
