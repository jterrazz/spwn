import { describe, test } from "vitest";

// TODO: requires Docker + running containers to spawn universes
describe.skip("visitor", () => {
  test("visitor runs ephemeral task in a universe", () => {});
  test("visitor has no Mind (no persistent state)", () => {});
  test("visitor without --universe flag fails", () => {});
  test("visitor with non-existent universe fails", () => {});
  test("visitor is fire-and-forget — no session persisted", () => {});
});
