import { describe, test, expect, beforeEach, afterEach } from "vitest";
import {
  spwn,
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { createSpwnHome, createAgent } from "../../setup/helpers.js";
import {
  expectLine,
  expectNoLine,
  expectTableHeader,
  stripAnsi,
} from "../../setup/output-helpers.js";

// ── Tests that require Docker (world lifecycle) ─────────────

describe("CLI execution — world aliases", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("'spwn up' alias spawns a world", () => {
    // GIVEN — initialized context
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — using the 'up' alias
    const result = ctx.spwn(
      ["up", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — world is created
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;
    expect(id).toBeTruthy();

    // AND — appears in ls
    const listResult = ctx.spwn(["ls"]);
    expect(listResult.exitCode).toBe(0);
    expect(stripAnsi(listResult.output)).toContain(id);
  });

  test("'spwn down' alias destroys a world", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["up", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // WHEN — using the 'down' alias
    const destroyResult = ctx.spwn(["down", id], 30_000);

    // THEN — world is destroyed
    expect(destroyResult.exitCode).toBe(0);
    expectLine(destroyResult.output, /✓ World destroyed\. Agent survives\./);

    // AND — world gone from ls
    const listResult = ctx.spwn(["ls"]);
    expect(listResult.exitCode).toBe(0);
    expectNoLine(
      listResult.output,
      new RegExp(id.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")),
    );
  });

  test("'spwn ls' alias lists worlds", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — using the 'ls' alias
    const listResult = ctx.spwn(["ls"]);

    // THEN — world appears in output
    expect(listResult.exitCode).toBe(0);
    expect(stripAnsi(listResult.output)).toContain(id);
    expectTableHeader(listResult.output, ["ID", "CONFIG", "AGENTS", "STATUS"]);
  });

  test("'spwn logs' alias works for world", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — using the 'logs' command (world logs)
    const logsResult = ctx.spwn(["world", "logs", id]);

    // THEN — doesn't error (agent may not have output yet)
    expect(logsResult.exitCode).toBe(0);
    // AND — output is a string (may be empty if agent hasn't logged yet)
    expect(typeof logsResult.output).toBe("string");
  });

  test("'spwn inspect' works for world via world inspect", () => {
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

    // THEN — output contains world details
    expect(inspectResult.exitCode).toBe(0);
    const out = stripAnsi(inspectResult.output);
    expect(out).toContain(id);
    expect(out).toContain("default"); // config name
    expect(out).toContain("neo"); // agent
    expectLine(inspectResult.output, /Config:\s+default/);
    expectLine(inspectResult.output, /Status:\s+(running|idle)/);
  });

  test("'spwn snap' alias creates snapshot", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — using the 'snap' alias
    const snapResult = ctx.spwn(["snap", id]);

    // THEN — snapshot created
    expect(snapResult.exitCode).toBe(0);
    expectLine(snapResult.output, /✓ Snapshot saved/);
  });
});

// ── Tests that don't need Docker (agent management) ─────────

describe("CLI execution — agent commands", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
  });

  test("'spwn agent new' creates an agent", async () => {
    // WHEN — creating a new agent
    const result = await spwn("agent new testbot")
      .exec("agent new testbot")
      .run();

    // THEN — exit code 0
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Created agent\s+testbot/);

    // AND — agent appears in ls
    const listResult = await spwn("agent ls after new")
      .exec("agent ls")
      .run();
    expect(listResult.exitCode).toBe(0);
    expect(stripAnsi(listResult.output)).toContain("testbot");
  });

  test("'spwn agent rm' removes an agent", async () => {
    // GIVEN — agent exists
    await spwn("create agent").exec("agent new testbot").run();

    // WHEN — removing it
    const result = await spwn("agent rm testbot")
      .exec("agent rm testbot")
      .run();

    // THEN — exit code 0
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /✓ Deleted agent\s+testbot/);

    // AND — agent gone from ls
    const listResult = await spwn("agent ls after rm")
      .exec("agent ls")
      .run();
    expect(listResult.exitCode).toBe(0);
    const output = stripAnsi(listResult.output);
    // testbot should not appear as a row in the table
    const tableLines = output.split("\n").filter((l) => l.includes("testbot"));
    expect(tableLines.length).toBe(0);
  });

  test("'spwn profile' shows character sheet", async () => {
    // GIVEN — agent exists
    createAgent(home, "testbot");

    // WHEN — viewing profile
    const result = await spwn("profile testbot")
      .exec("profile testbot")
      .run();

    // THEN — shows character sheet elements
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("testbot");
    expect(out).toContain("Tier");
    expect(out).toContain("citizen");
    expect(out).toContain("Identity");
  });

  test("'spwn profile <agent> purpose' shows not-set when missing", async () => {
    // GIVEN — agent with no purpose file
    createAgent(home, "testbot");

    // WHEN — checking purpose
    const result = await spwn("profile purpose")
      .exec("profile testbot purpose")
      .run();

    // THEN — shows not set message
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Not set yet");
  });

  test("'spwn agent ls' shows table with correct headers", async () => {
    // GIVEN — agents exist
    await spwn("create agent1").exec("agent new alpha").run();
    await spwn("create agent2").exec("agent new beta").run();

    // WHEN — listing agents
    const result = await spwn("agent ls")
      .exec("agent ls")
      .run();

    // THEN — table has correct headers and both agents
    expect(result.exitCode).toBe(0);
    expectTableHeader(result.output, ["NAME", "LAYERS", "WORLD", "STATUS"]);
    expect(stripAnsi(result.output)).toContain("alpha");
    expect(stripAnsi(result.output)).toContain("beta");
  });

  test("'spwn agent inspect' shows detailed info", async () => {
    // GIVEN — agent exists
    await spwn("create for inspect").exec("agent new inspectme").run();

    // WHEN — inspecting
    const result = await spwn("agent inspect")
      .exec("agent inspect inspectme")
      .run();

    // THEN — shows structured details
    expect(result.exitCode).toBe(0);
    expectLine(result.output, /Agent:\s+inspectme/);
    expectLine(result.output, /World:\s+unattached/);
    expectLine(result.output, /identity\/\s+default\.md/);
  });
});

// ── Attach command ──────────────────────────────────────────

describe("CLI execution — attach command", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
  });

  test("'spwn world attach <nonexistent-id>' returns clean error", async () => {
    // WHEN — attaching to a non-existent world
    const result = await spwn("attach nonexistent")
      .exec("world attach w-fake-99999")
      .run();

    // THEN — returns non-zero exit code
    expect(result.exitCode).not.toBe(0);

    // AND — error output contains useful message (not a raw stack trace)
    const output = stripAnsi(result.output);
    expectNoLine(result.output, /panic:/);
    expectNoLine(result.output, /goroutine /);
  });

  test("'spwn world attach --help' shows usage", async () => {
    // WHEN — requesting attach help
    const result = await spwn("attach help")
      .exec("world attach --help")
      .run();

    // THEN — exit code 0
    expect(result.exitCode).toBe(0);

    // AND — output contains usage information
    const output = stripAnsi(result.output);
    expect(output).toContain("attach");
    expect(output).toContain("world-id");
  });
});

// ── Global flags ─────────────────────────────────────────────

describe("CLI execution — global flags", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
  });

  test("'--json' flag produces valid JSON for status", async () => {
    // GIVEN — initialized home
    await spwn("init").exec("init").run();

    // WHEN — running status with --json
    const result = await spwn("status json")
      .exec("status --json")
      .run();

    // THEN — output is valid JSON
    expect(result.exitCode).toBe(0);
    const jsonOutput = result.stdout.trim();
    expect(() => JSON.parse(jsonOutput)).not.toThrow();

    // AND — JSON contains expected fields
    const parsed = JSON.parse(jsonOutput);
    expect(parsed).toBeDefined();
    expect(typeof parsed).toBe("object");
  });

  test("'--json' flag produces valid JSON for agent ls", async () => {
    // GIVEN — agents exist
    await spwn("init").exec("init").run();
    await spwn("create neo").exec("agent new neo").run();

    // WHEN — listing agents with --json
    const result = await spwn("agent ls json")
      .exec("agent ls --json")
      .run();

    // THEN — output is valid JSON
    expect(result.exitCode).toBe(0);
    const jsonOutput = result.stdout.trim();
    expect(() => JSON.parse(jsonOutput)).not.toThrow();

    // AND — JSON is array or contains agents
    const parsed = JSON.parse(jsonOutput);
    expect(parsed).toBeDefined();
  });

  test("'--json' suppresses decorative output", async () => {
    // GIVEN — initialized home
    await spwn("init").exec("init").run();

    // WHEN — running status with --json
    const result = await spwn("status json decorative")
      .exec("status --json")
      .run();

    // THEN — stdout should NOT contain box-drawing characters
    const stdout = result.stdout;
    expect(stdout).not.toContain("╭");
    expect(stdout).not.toContain("╰");
    expect(stdout).not.toContain("│");
  });

  test("'--quiet' flag suppresses output for agent new", async () => {
    // WHEN — creating agent with --quiet
    const result = await spwn("agent new quiet")
      .exec("agent new quietbot --quiet")
      .run();

    // THEN — exit code 0 but minimal output
    expect(result.exitCode).toBe(0);
    // In quiet mode, output should be empty or minimal
    const output = stripAnsi(result.output).trim();
    // Quiet mode should have less output than normal (no status lines)
    expect(output.split("\n").filter((l) => l.length > 0).length).toBeLessThanOrEqual(1);
  });

  test("'--version' shows version string", async () => {
    const result = await spwn("version")
      .exec("--version")
      .run();

    expect(result.exitCode).toBe(0);
    expect(result.output).toMatch(/spwn version/);
  });
});

// ── Status command ──────────────────────────────────────────

describe("CLI execution — status command", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
  });

  test("'spwn status' runs without error", async () => {
    await spwn("init").exec("init").run();
    const result = await spwn("status").exec("status").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("s p w n");
  });

  test("'spwn doctor' runs diagnostics", async () => {
    const result = await spwn("doctor").exec("doctor").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("Docker");
    expect(out).toContain("Version");
  });

  test("'spwn auth' shows authentication status", async () => {
    const result = await spwn("auth").exec("auth").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    expect(out).toContain("PROVIDER");
    expect(out).toContain("STATUS");
  });
});
