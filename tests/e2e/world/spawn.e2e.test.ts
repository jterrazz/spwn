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
    expectLine(result.output, /✓ Loaded config\s+default/);
    expectLine(result.output, /✓ Docker connected/);
    expectLine(result.output, /✓ Validated agent\s+neo/);
    expectLine(result.output, /✓ Mounted mind\s+neo → \/mind/);
    expectLine(result.output, /✓ (Built image|Image ready)\s+spwn-test:latest/);
    expectLine(result.output, /✓ Credentials\s+/);
    expectLine(result.output, /✓ Created container\s+w-\w+-\d{5}/);
    expectLine(result.output, /✓ Probed elements\s+\d+ verified/);
    expectLine(result.output, /✓ Generated physics\s+physics\.md, faculties\.md/);
    expectLine(result.output, /✓ Agent is alive\./);
    const id = parseWorldId(result.output)!;
    expect(id).toBeTruthy();
    expect(id).toMatch(/^w-\w+-\d{5}$/);

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
    expectLine(result.output, /✓ Created container\s+w-myconfig-\d{5}/);
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

  test("fails with non-existent config — clean error, no usage dump", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning with a non-existent config
    const result = ctx.spwn(
      ["world", "-c", "nonexistent", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — exits with error and shows actionable hint
    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /✗ Config failed/);
    expectLine(result.output, /spwn init/);

    // AND — does NOT dump full usage
    expectNoLine(result.output, /Available Commands:/);
    expectNoLine(result.output, /Global Flags:/);
  });

  test("fails with non-existent agent — shows init hint", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const result = ctx.spwn(
      ["world", "--agent", "ghost", "-w", ctx.home],
      60_000,
    );

    expect(result.exitCode).not.toBe(0);
    expectLine(result.output, /✗ Spawn failed/);
    expectLine(result.output, /spwn agent init ghost/);
    expectNoLine(result.output, /Available Commands:/);
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

    for (const layer of ["identity", "skills", "memory/knowledge", "memory/playbooks", "memory/journal", "sessions"]) {
      ctx.universe(id).toHaveDirectory(`/mind/${layer}`);
    }
  });

  test("container has Claude trust pre-approved", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--no-agent", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(result.output)!;

    // Verify .claude.json has trust accepted for /workspace
    const claudeJson = ctx.universe(id).readFile("/home/spwn/.claude.json");
    const config = JSON.parse(claudeJson);
    expect(config.hasCompletedOnboarding).toBe(true);
    expect(config.projects["/workspace"].hasTrustDialogAccepted).toBe(true);

    // Verify settings.json skips dangerous mode prompt
    const settings = ctx.universe(id).readFile("/home/spwn/.claude/settings.json");
    const settingsConfig = JSON.parse(settings);
    expect(settingsConfig.skipDangerousModePermissionPrompt).toBe(true);
  });

  test("container does NOT mount host ~/.claude directory", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--no-agent", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(result.output)!;

    // The settings.json should be our minimal config, not the host's (which has hooks, plugins, etc.)
    const settings = ctx.universe(id).readFile("/home/spwn/.claude/settings.json");
    const config = JSON.parse(settings);
    // Host config has hooks.PreToolUse — container config should NOT
    expect(config.hooks).toBeUndefined();
    expect(config.enabledPlugins).toBeUndefined();
  });

  test("default mode is detached (no --interactive needed)", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // Spawn WITHOUT --detach or --interactive — should still be detached
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    expect(result.exitCode).toBe(0);
    // The mock agent runs and exits, world goes idle
    expectLine(result.output, /Agent spawned\s+detached/);
    // Should show talk hint
    expectLine(result.output, /Talk: spwn agent talk neo/);
  });

  test("world ID uses planet name instead of 'default'", () => {
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(result.output)!;

    // ID should NOT contain "default"
    expect(id).not.toContain("default");
    // ID should match w-{planet}-{digits} format
    expect(id).toMatch(/^w-[a-z]+-\d{5}$/);
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
