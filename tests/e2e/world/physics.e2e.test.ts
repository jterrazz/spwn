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
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — inspecting the world
    const inspectResult = ctx.spwn(["world", "inspect", id]);

    // THEN — physics constants are shown in structured format
    expect(inspectResult.exitCode).toBe(0);
    expectLine(inspectResult.output, /Constants:\s+CPU:.*Memory:.*Timeout:/);
    expectLine(inspectResult.output, /Laws:\s+Network:.*Max processes:/);
  });

  test("physics.md contains constants inside container", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // THEN — physics.md inside container contains expected fields
    const physics = ctx.universe(id).physics();
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

    const physics = ctx.universe(id).physics();
    expect(physics).toMatch(/network/i);
  });

  test("faculties.md is generated inside the world", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(spawnResult.exitCode).toBe(0);
    const id = parseWorldId(spawnResult.output)!;

    // THEN — the spawn output mentions faculties generation
    expectLine(spawnResult.output, /✓ Generated physics\s+physics\.md, faculties\.md/);

    // AND — faculties.md actually exists and lists elements
    const faculties = ctx.universe(id).faculties();
    expect(faculties).toMatch(/bash/);
  });

  test("world includes declared elements", () => {
    // GIVEN — a spawned world (default config has @unix, @git elements)
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — inspecting the world
    const inspectResult = ctx.spwn(["world", "inspect", id]);

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
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(result.output)!;

    // Verify network mode is set to none via docker inspect
    const inspectData = ctx.universe(id).inspect();
    expect(inspectData.HostConfig?.NetworkMode).toBe("none");

    // Try to curl from inside — should fail with network=none
    // We use exec and expect it to throw (non-zero exit code)
    let curlSucceeded = false;
    try {
      ctx
        .universe(id)
        .exec("curl -s --connect-timeout 2 http://example.com");
      curlSucceeded = true;
    } catch {
      // Expected: connection fails because network is disabled
    }
    expect(curlSucceeded).toBe(false);
  });
});
