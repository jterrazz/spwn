import { describe, test, expect, afterEach } from "vitest";
import { readFileSync, existsSync } from "node:fs";
import { join } from "node:path";
import {
  createTestContext,
  parseUniverseId,
  type TestContext,
} from "../../setup/spwn.specification.js";

describe("state management", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("universe state persists in state.json", async () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // THEN — state.json exists and contains the universe
    const statePath = join(ctx.home, "state.json");
    expect(existsSync(statePath)).toBe(true);
    const state = JSON.parse(readFileSync(statePath, "utf-8"));
    expect(Array.isArray(state)).toBe(true);
    const entry = state.find((u: { id: string }) => u.id === id);
    expect(entry).toBeTruthy();
    expect(entry.agent).toBe("neo");
    expect(entry.backend).toBe("docker");
  });

  test("destroy updates state file", async () => {
    // GIVEN — a spawned and destroyed universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;
    ctx.spwn(["universe", "destroy", id], 30_000);

    // THEN — state.json no longer contains the universe
    const statePath = join(ctx.home, "state.json");
    const state = JSON.parse(readFileSync(statePath, "utf-8"));
    const entry = state.find((u: { id: string }) => u.id === id);
    expect(entry).toBeFalsy();
  });

  test("state tracks active universes across list calls", async () => {
    // GIVEN — a spawned universe
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const spawnResult = ctx.spwn(
      ["universe", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const id = parseUniverseId(spawnResult.output)!;

    // WHEN — listing universes multiple times
    const list1 = ctx.spwn(["universe", "list"]);
    const list2 = ctx.spwn(["universe", "list"]);

    // THEN — both calls show the same universe
    expect(list1.output).toContain(id);
    expect(list2.output).toContain(id);
  });
});
