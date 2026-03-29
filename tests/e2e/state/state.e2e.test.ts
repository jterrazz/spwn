import { describe, test } from "vitest";

// TODO: requires Docker + running containers to spawn universes
describe.skip("state management", () => {
  test("universe state persists across list calls", () => {});
  test("destroy updates state file", () => {});
  test("claw state tracks active universes", () => {});
  test("agent state persists across commands", () => {});
  test("concurrent universe spawns get unique IDs", () => {});
});
