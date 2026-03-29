import { describe, test } from "vitest";

// TODO: requires Docker + running containers to spawn universes
describe.skip("config cascade", () => {
  test("org.yaml provides organization-level defaults", () => {});
  test("universe.yaml overrides org.yaml", () => {});
  test("life.yaml overrides universe.yaml for agent", () => {});
  test("missing org.yaml triggers initialization hint", () => {});
  test("multiple universe configs coexist", () => {});
});
