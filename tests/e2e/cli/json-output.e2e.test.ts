import { describe, test, expect, beforeEach, afterEach } from "vitest";
import {
  spwn,
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";
import { stripAnsi } from "../../setup/output-helpers.js";

// ── --json flag verification for commands without Docker ─────

describe("--json output", () => {
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

  test("status --json outputs valid JSON", async () => {
    // GIVEN — initialized home
    await spwn("init").exec("init").run();

    // WHEN — running status with --json
    const result = await spwn("status json")
      .exec("status --json")
      .run();

    // THEN — exit code 0 and valid JSON
    expect(result.exitCode).toBe(0);
    const jsonStr = result.stdout.trim();
    let parsed: unknown;
    expect(() => {
      parsed = JSON.parse(jsonStr);
    }).not.toThrow();

    // AND — JSON is an object with expected structure
    expect(typeof parsed).toBe("object");
    expect(parsed).not.toBeNull();
  });

  test("agent ls --json outputs valid JSON", async () => {
    // GIVEN — agents exist
    await spwn("init").exec("init").run();
    await spwn("create neo").exec("agent new neo").run();
    await spwn("create trinity").exec("agent new trinity").run();

    // WHEN — listing agents with --json
    const result = await spwn("agent ls json")
      .exec("agent ls --json")
      .run();

    // THEN — valid JSON output
    expect(result.exitCode).toBe(0);
    const jsonStr = result.stdout.trim();
    let parsed: unknown;
    expect(() => {
      parsed = JSON.parse(jsonStr);
    }).not.toThrow();

    // AND — JSON is array or contains agents field
    expect(parsed).toBeDefined();
    if (Array.isArray(parsed)) {
      expect(parsed.length).toBeGreaterThanOrEqual(2);
    }
  });

  test("status --json has no ANSI codes in stdout", async () => {
    // GIVEN — initialized home
    await spwn("init").exec("init").run();

    // WHEN — running status with --json
    const result = await spwn("status json ansi")
      .exec("status --json")
      .run();

    // THEN — stdout has no ANSI escape codes
    expect(result.exitCode).toBe(0);
    const raw = result.stdout;
    // ANSI escape code regex
    expect(raw).not.toMatch(/\x1B\[[0-9;]*[a-zA-Z]/);
  });

  test("status --json has no box-drawing characters in stdout", async () => {
    // GIVEN — initialized home
    await spwn("init").exec("init").run();

    // WHEN — running with --json
    const result = await spwn("status json boxes")
      .exec("status --json")
      .run();

    // THEN — stdout has no decorative characters
    expect(result.exitCode).toBe(0);
    const stdout = result.stdout;
    expect(stdout).not.toContain("╭");
    expect(stdout).not.toContain("╮");
    expect(stdout).not.toContain("╰");
    expect(stdout).not.toContain("╯");
    expect(stdout).not.toContain("─");
    // Box chars might be in stderr for messages, but stdout should be pure JSON
  });

  test("agent ls --json has no table formatting in stdout", async () => {
    // GIVEN — agents exist
    await spwn("init").exec("init").run();
    await spwn("create neo").exec("agent new neo").run();

    // WHEN — listing with --json
    const result = await spwn("agent ls json fmt")
      .exec("agent ls --json")
      .run();

    // THEN — stdout is pure JSON, no table headers
    expect(result.exitCode).toBe(0);
    const stdout = result.stdout;
    // Table headers should NOT be in JSON output
    expect(stdout).not.toContain("NAME");
    expect(stdout).not.toContain("LAYERS");
    expect(stdout).not.toContain("STATUS");
  });
});

// ── --json with Docker (world commands) ─────────────────────

describe("--json output with worlds", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("ls --json outputs valid JSON with world data", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // WHEN — listing worlds with --json
    const result = ctx.spwn(["ls", "--json"]);

    // THEN — valid JSON containing world info
    expect(result.exitCode).toBe(0);
    const jsonStr = result.stdout.trim();
    let parsed: unknown;
    expect(() => {
      parsed = JSON.parse(jsonStr);
    }).not.toThrow();

    // AND — contains the world ID somewhere in the JSON
    expect(JSON.stringify(parsed)).toContain(id);
  });

  test("world list --json outputs valid JSON", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseWorldId(spawnResult.output)!;

    // WHEN — listing with explicit world list --json
    const result = ctx.spwn(["world", "list", "--json"]);

    // THEN — valid JSON
    expect(result.exitCode).toBe(0);
    const jsonStr = result.stdout.trim();
    expect(() => JSON.parse(jsonStr)).not.toThrow();
  });

  test("--json world list has no decorative output in stdout", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // WHEN — listing worlds with --json
    const result = ctx.spwn(["ls", "--json"]);

    // THEN — no table headers or box chars in stdout
    expect(result.exitCode).toBe(0);
    const stdout = result.stdout;
    expect(stdout).not.toMatch(/\x1B\[[0-9;]*[a-zA-Z]/);
    expect(stdout).not.toContain("╭");
    expect(stdout).not.toContain("╰");
  });
});
