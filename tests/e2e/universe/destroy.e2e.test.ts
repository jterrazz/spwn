import { describe, test, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { spwn, mockApi } from "../../setup/spwn.specification.js";
import { createSpwnHome, createUniverseConfig, createOrgManifest } from "../../setup/helpers.js";
import { noop } from "../../setup/mock-api/handlers.js";

describe("universe destroy", () => {
  let home: string;

  beforeAll(async () => {
    await mockApi.start();
  });

  afterAll(async () => {
    await mockApi.stop();
  });

  beforeEach(() => {
    // GIVEN — a fresh SPWN_HOME with default config
    home = createSpwnHome();
    createOrgManifest(home);
    createUniverseConfig(home, "default");
    mockApi.reset();
    mockApi.onChat(noop);
  });

  test("destroys a running universe", async () => {
    // GIVEN — a spawned universe
    const spawnResult = await spwn("spawn for destroy")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — destroying the universe
    const result = await spwn("destroy universe")
      .exec(`universe destroy ${universeId}`)
      .run();

    // THEN — universe is destroyed successfully
    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("destroyed");
  });

  test("destroy removes universe from list", async () => {
    // GIVEN — a spawned universe that gets destroyed
    const spawnResult = await spwn("spawn for list check")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    await spwn("destroy for list check")
      .exec(`universe destroy ${universeId}`)
      .run();

    // WHEN — listing universes
    const listResult = await spwn("list after destroy")
      .exec("universe list")
      .run();

    // THEN — the destroyed universe is no longer listed
    expect(listResult.stdout).not.toContain(universeId);
  });

  test("destroy non-existent universe fails", async () => {
    // WHEN — destroying a universe that does not exist
    const result = await spwn("destroy missing")
      .exec("universe destroy u-nonexistent-00000")
      .run();

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
    expect(result.stderr).toContain("not found");
  });
});
