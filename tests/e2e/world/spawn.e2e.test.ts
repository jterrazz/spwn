import { describe, test, expect, afterEach } from "vitest";
import { writeFileSync } from "node:fs";
import { join } from "node:path";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine, expectNoLine } from "../../setup/output-helpers.js";

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

    // THEN — exits successfully with structured status lines
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Mounted mind\s+neo → \/mind/);
    expectLine(result.output, /✓ Built image\s+spwn-test:latest/);
    expectLine(result.output, /✓ Spawned world\s+w-default-\d{5}/);
    expectLine(result.output, /✓ Generated faculties\s+physics\.md, faculties\.md/);
    expectLine(result.output, /✓ Agent is alive\./);
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
    expectLine(result.output, /✓ Spawned world\s+w-myconfig-\d{5}/);
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

    // THEN — the spawned world appears in the table
    expect(listResult.exitCode).toBe(0);
    expectLine(listResult.output, new RegExp(id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")));

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
    expect(physics).toMatch(/CPU/);
    expect(physics).toMatch(/Memory/);
    expect(physics).toMatch(/Timeout/);
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
    expect(faculties).toMatch(/bash/);
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

  test("workspace files visible inside container", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // Create a test file in workspace
    writeFileSync(join(ctx.home, "test-file.txt"), "hello from workspace");

    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(result.output)!;

    // Verify file exists inside container
    ctx.universe(id).toHaveFile("/workspace/test-file.txt", "hello from workspace");
  });

  test("Mind has all 6 layers inside container", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(result.output)!;

    for (const layer of ["personas", "skills", "knowledge", "playbooks", "journal", "sessions"]) {
      ctx.universe(id).toHaveDirectory(`/mind/${layer}`);
    }
  });

  test("--no-agent spawns world without agent", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--no-agent", "-w", ctx.home],
      60_000,
    );

    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;
    expect(id).toBeTruthy();
    ctx.universe(id).toBeRunning();

    // World exists in state
    ctx.state().hasWorld(id);
  });
});
