import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("gate bridge", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("gate bridges element access into container", async () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning with a gate bridge
    const spawnResult = ctx.spwn(
      [
        "universe",
        "--agent",
        "neo",
        "--gate",
        "bash:as:exec",
        "-w",
        ctx.home,
      ],
      60_000,
    );

    // THEN — the output confirms gate was bridged
    expect(spawnResult.exitCode).toBe(0);
    expect(spawnResult.output).toContain("Bridged gate");
    expect(spawnResult.output).toContain("1 element(s)");
  });

  test("spawn without gate does not mention bridging", async () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning without gate
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — no gate bridging in output
    expect(spawnResult.exitCode).toBe(0);
    expect(spawnResult.output).not.toContain("Bridged gate");
  });

  test("faculties.md reflects bridged elements", async () => {
    // GIVEN — a universe with gate bridge
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "universe",
        "--agent",
        "neo",
        "--gate",
        "bash:as:exec",
        "-w",
        ctx.home,
      ],
      60_000,
    );

    // THEN — faculties were generated (they include bridged elements)
    expect(spawnResult.exitCode).toBe(0);
    expect(spawnResult.output).toContain("Generated faculties");
  });
});
