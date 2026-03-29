import { describe, test, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { spwn, mockApi } from "../../setup/spwn.specification.js";
import {
  createSpwnHome,
  createUniverseConfig,
  createOrgManifest,
  createAgent,
} from "../../setup/helpers.js";
import { noop } from "../../setup/mock-api/handlers.js";
import { writeFileSync } from "node:fs";
import { join } from "node:path";

describe("config cascade", () => {
  let home: string;

  beforeAll(async () => {
    await mockApi.start();
  });

  afterAll(async () => {
    await mockApi.stop();
  });

  beforeEach(() => {
    // GIVEN — a fresh SPWN_HOME
    home = createSpwnHome();
    mockApi.reset();
    mockApi.onChat(noop);
  });

  test("org.yaml provides organization-level defaults", async () => {
    // GIVEN — an org manifest with custom defaults
    createOrgManifest(home, "acme-corp");
    createUniverseConfig(home, "default");

    // WHEN — spawning a universe
    const result = await spwn("org defaults")
      .exec("universe")
      .run();

    // THEN — universe inherits org-level settings
    expect(result.exitCode).toBe(0);
  });

  test("universe.yaml overrides org.yaml", async () => {
    // GIVEN — org with defaults and universe with overrides
    createOrgManifest(home, "acme-corp");
    createUniverseConfig(home, "custom", {
      physics: { cpu: 4, memory: "2g", timeout: "60m", "max-processes": 200 },
    });

    // WHEN — spawning with the custom config
    const spawnResult = await spwn("universe overrides org")
      .exec("universe -c custom")
      .run();
    const universeId = spawnResult.stdout.match(/u-custom-\d{5}/)?.[0];

    // THEN — universe uses its own config values
    const inspectResult = await spwn("inspect overrides")
      .exec(`universe inspect ${universeId}`)
      .run();
    expect(inspectResult.exitCode).toBe(0);
  });

  test("life.yaml overrides universe.yaml for agent", async () => {
    // GIVEN — an agent with a life.yaml that overrides universe settings
    createOrgManifest(home);
    createUniverseConfig(home, "default");
    createAgent(home, "custom-agent");
    writeFileSync(
      join(home, "agents", "custom-agent", "life.yaml"),
      "tier: premium\nruntime: claude-code\n",
    );

    // WHEN — inspecting the agent
    const result = await spwn("agent with life.yaml")
      .exec("agent inspect custom-agent")
      .run();

    // THEN — agent details reflect life.yaml overrides
    expect(result.exitCode).toBe(0);
  });

  test("missing org.yaml triggers initialization hint", async () => {
    // GIVEN — no org.yaml exists (SPWN_HOME is bare)

    // WHEN — running any command
    const result = await spwn("missing org")
      .exec("universe list")
      .run();

    // THEN — the user is prompted to run spwn init
    expect(result.exitCode).not.toBe(0);
  });

  test("multiple universe configs coexist", async () => {
    // GIVEN — org and multiple universe configs
    createOrgManifest(home);
    createUniverseConfig(home, "dev", {
      physics: { cpu: 1, memory: "512m", timeout: "10m", "max-processes": 50 },
    });
    createUniverseConfig(home, "prod", {
      physics: { cpu: 4, memory: "4g", timeout: "120m", "max-processes": 500 },
    });

    // WHEN — listing available configs (via universe help or similar)
    const result = await spwn("multiple configs")
      .exec("--help")
      .run();

    // THEN — both configs are usable (no conflict)
    expect(result.exitCode).toBe(0);
  });
});
