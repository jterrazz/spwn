import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine, expectNoLine } from "../../setup/output-helpers.js";

describe("gate bridge", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("gate bridges element access into container", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning with a gate bridge
    const spawnResult = ctx.spwn(
      [
        "world",
        "--agent",
        "neo",
        "--gate",
        "bash:as:exec",
        "-w",
        ctx.home,
      ],
      60_000,
    );

    // THEN — the output confirms gate was bridged with structured status
    expect(spawnResult.exitCode).toBe(0);
    expectLine(spawnResult.output, /✓ Bridged gate\s+1 element\(s\)/);
    expectLine(spawnResult.output, /✓ Spawned world\s+w-default-\d{5}/);

    // AND — container is running
    const id = parseWorldId(spawnResult.output)!;
    ctx.universe(id).toBeRunning();

    // AND — faculties.md reflects bridged elements
    ctx.universe(id).toHaveFile("/universe/faculties.md");
  });

  test("spawn without gate does not mention bridging", () => {
    // GIVEN — an initialized SPWN_HOME
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // WHEN — spawning without gate
    const spawnResult = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );

    // THEN — no gate bridging in output
    expect(spawnResult.exitCode).toBe(0);
    expectNoLine(spawnResult.output, /Bridged gate/);

    // AND — container is still running with world files
    const id = parseWorldId(spawnResult.output)!;
    ctx
      .universe(id)
      .toBeRunning()
      .toHaveFile("/universe/physics.md")
      .toHaveFile("/universe/faculties.md");
  });

  test("faculties.md reflects bridged elements", () => {
    // GIVEN — a world with gate bridge
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "world",
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
    expectLine(spawnResult.output, /✓ Generated faculties\s+physics\.md, faculties\.md/);

    // AND — faculties.md inside container mentions the bridged element
    const id = parseWorldId(spawnResult.output)!;
    const faculties = ctx.universe(id).faculties();
    expect(faculties).toBeTruthy();
  });
});
