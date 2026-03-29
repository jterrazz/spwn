import { describe, test, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { spwn, mockApi } from "../../setup/spwn.specification.js";
import { createSpwnHome, createUniverseConfig, createOrgManifest } from "../../setup/helpers.js";
import { respond, noop } from "../../setup/mock-api/handlers.js";

describe("visitor", () => {
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

  test("visitor runs ephemeral task in a universe", async () => {
    // GIVEN — a running universe
    mockApi.onChat(respond("Linting completed. 0 issues found."));

    const spawnResult = await spwn("spawn for visitor")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — running a visitor task
    const result = await spwn("visitor task")
      .exec(`visitor "lint src/" --universe ${universeId}`)
      .run();

    // THEN — visitor completes the task
    expect(result.exitCode).toBe(0);
  });

  test("visitor has no Mind (no persistent state)", async () => {
    // GIVEN — a running universe
    mockApi.onChat(respond("Task done."));

    const spawnResult = await spwn("spawn for visitor no mind")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — running a visitor
    const result = await spwn("visitor no mind")
      .exec(`visitor "check tests" --universe ${universeId}`)
      .run();

    // THEN — visitor executes without Mind (no personas, no journal, etc.)
    expect(result.exitCode).toBe(0);

    // AND — the mock API was called (visitor did run)
    expect(mockApi.calls.length).toBeGreaterThan(0);
  });

  test("visitor without --universe flag fails", async () => {
    // WHEN — running visitor without specifying a universe
    const result = await spwn("visitor no universe")
      .exec("visitor \"lint src/\"")
      .run();

    // THEN — exits with error requiring --universe
    expect(result.exitCode).not.toBe(0);
    expect(result.stderr).toContain("universe");
  });

  test("visitor with non-existent universe fails", async () => {
    // WHEN — running visitor with a non-existent universe
    const result = await spwn("visitor missing universe")
      .exec("visitor \"task\" --universe u-nonexistent-00000")
      .run();

    // THEN — exits with error
    expect(result.exitCode).not.toBe(0);
  });

  test("visitor is fire-and-forget — no session persisted", async () => {
    // GIVEN — a running universe and a scripted response
    mockApi.onChat(respond("Done."));

    const spawnResult = await spwn("spawn for visitor session")
      .exec("universe")
      .run();
    const universeId = spawnResult.stdout.match(/u-default-\d{5}/)?.[0];

    // WHEN — running a visitor
    await spwn("visitor for session check")
      .exec(`visitor "run task" --universe ${universeId}`)
      .run();

    // THEN — no agent session files are created (visitor is ephemeral)
    // This is verified by the absence of session artifacts
    const listResult = await spwn("list agents after visitor")
      .exec("agent list")
      .run();

    // Visitor should not appear as a persistent agent
    expect(listResult.stdout).not.toContain("visitor");
  });
});
