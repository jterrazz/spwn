import { describe, test } from "vitest";

// TODO: requires Docker + running containers to spawn universes
describe.skip("universe spawn", () => {
  test("spawns a universe with default config", () => {});
  test("spawns a universe with named config via -c flag", () => {});
  test("spawns a universe with governor agent", () => {});
  test("spawns a universe with workspace mount", () => {});
  test("fails with non-existent config", () => {});
  test("universe ID format is u-{name}-{5digits}", () => {});
});
