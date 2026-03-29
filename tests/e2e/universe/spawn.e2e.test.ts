import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("universe spawn", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("spawns a universe with default config", async () => {
    // GIVEN — an initialized SPWN_HOME with agent
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning a universe
    const result = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — exits successfully and outputs a universe ID
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("Spawned universe");
    const id = parseUniverseId(result.output);
    expect(id).toBeTruthy();
    expect(id).toMatch(/^u-default-\d{5}$/);
  });

  test("spawns a universe with named config via -c flag", async () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init", "myconfig"]);

    // WHEN — spawning with a named config
    const result = ctx.spwn(
      ["universe", "-c", "myconfig", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — exits successfully
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("Spawned universe");
    const id = parseUniverseId(result.output);
    expect(id).toMatch(/^u-myconfig-\d{5}$/);
  });

  test("universe ID format is u-{name}-{5digits}", async () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning a universe
    const result = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — the ID matches the expected format
    const id = parseUniverseId(result.output);
    expect(id).toBeTruthy();
    expect(id).toMatch(/^u-\w+-\d{5}$/);
  });

  test("spawned universe appears in list", async () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output);
    expect(id).toBeTruthy();

    // WHEN — listing universes
    const listResult = ctx.spwn(["universe", "list"]);

    // THEN — the spawned universe appears
    expect(listResult.exitCode).toBe(0);
    expect(listResult.output).toContain(id!);
  });

  test("fails with non-existent config", async () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning with a non-existent config
    const result = ctx.spwn(
      ["universe", "-c", "nonexistent", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
  });
});
