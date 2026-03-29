import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("world spawn", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("creates a running Docker container", () => {
    // GIVEN — an initialized SPWN_HOME with agent
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning a world
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — exits successfully and outputs a world ID
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("Spawned world");
    const id = parseWorldId(result.output)!;
    expect(id).toBeTruthy();
    expect(id).toMatch(/^w-default-\d{5}$/);

    // AND — the container is actually running
    ctx.universe(id).toBeRunning();

    // AND — has /universe directory with physics + faculties
    ctx
      .universe(id)
      .toHaveFile("/universe/physics.md")
      .toHaveFile("/universe/faculties.md");
  });

  test("spawns a world with named config via -c flag", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init", "myconfig"]);

    // WHEN — spawning with a named config
    const result = ctx.spwn(
      ["world", "-c", "myconfig", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — exits successfully with correct ID prefix
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("Spawned world");
    const id = parseWorldId(result.output)!;
    expect(id).toMatch(/^w-myconfig-\d{5}$/);

    // AND — container is running
    ctx.universe(id).toBeRunning();
  });

  test("world ID format is w-{name}-{5digits}", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning a world
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — the ID matches the expected format
    const id = parseWorldId(result.output);
    expect(id).toBeTruthy();
    expect(id).toMatch(/^w-\w+-\d{5}$/);
  });

  test("spawned world appears in list", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // WHEN — listing worlds
    const listResult = ctx.spwn(["world", "list"]);

    // THEN — the spawned world appears
    expect(listResult.exitCode).toBe(0);
    expect(listResult.output).toContain(id);

    // AND — state.json tracks it
    ctx.state().hasWorld(id).hasAgent(id, "neo");
  });

  test("fails with non-existent config", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning with a non-existent config
    const result = ctx.spwn(
      ["world", "-c", "nonexistent", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
  });

  test("physics.md contains declared constants", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(result.output)!;

    // Read physics.md from inside the container
    const physics = ctx.universe(id).physics();
    expect(physics).toContain("CPU");
    expect(physics).toContain("Memory");
    expect(physics).toContain("Timeout");
  });

  test("faculties.md lists verified elements", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(result.output)!;

    const faculties = ctx.universe(id).faculties();
    expect(faculties).toContain("bash");
  });

  test("container removed after destroy", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(result.output)!;

    ctx.universe(id).toBeRunning();
    ctx.spwn(["world", "destroy", id], 30_000);
    ctx.universe(id).toNotExist();
  });
});
