import { describe, test, expect, afterEach } from "vitest";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { createAgent } from "../../setup/helpers.js";

describe("colony multi-agent", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("spawn universe with governor agent", () => {
    // GIVEN — two agents: neo as citizen, morpheus as governor
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);

    // WHEN — spawning with governor
    const spawnResult = ctx.spwn(
      [
        "universe",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );

    // THEN — succeeds and mounts both minds
    expect(spawnResult.exitCode).toBe(0);
    expect(spawnResult.output).toContain("Spawned universe");
    expect(spawnResult.output).toContain("neo");
    expect(spawnResult.output).toContain("morpheus");

    // AND — container is running with universe files
    const id = parseUniverseId(spawnResult.output)!;
    ctx
      .universe(id)
      .toBeRunning()
      .toHaveFile("/universe/physics.md")
      .toHaveFile("/universe/faculties.md");

    // AND — both agents exist on host
    ctx.mind("neo").exists();
    ctx.mind("morpheus").exists();
  });

  test("destroying universe with governor cleans up", () => {
    // GIVEN — a universe with governor
    ctx = createTestContext();
    createAgent(ctx.home, "morpheus");
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      [
        "universe",
        "--agent",
        "neo",
        "--governor",
        "morpheus",
        "-w",
        ctx.home,
      ],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;
    expect(id).toBeTruthy();

    // Verify running before destroy
    ctx.universe(id).toBeRunning();

    // WHEN — destroying
    const destroyResult = ctx.spwn(["universe", "destroy", id], 30_000);

    // THEN — cleans up
    expect(destroyResult.exitCode).toBe(0);
    expect(destroyResult.output).toContain("Universe destroyed");

    // AND — container is gone
    ctx.universe(id).toNotExist();

    // AND — list is empty
    const listResult = ctx.spwn(["universe", "list"]);
    expect(listResult.output).not.toContain(id);

    // AND — state no longer has it
    ctx.state().noUniverse(id);
  });
});
