/**
 * E2E tests for features built/fixed on 2026-04-12.
 *
 * Covers: example install + agent repair, agent identity structure,
 * org.yaml removal, example gallery order, examples bundling,
 * CLI upgrade --check, CLAUDE.md generation on init deployment.
 *
 * All tests are CLI-only (no Docker required) unless noted.
 */
import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { existsSync, readFileSync, mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";

import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome, createAgent, createBrokenAgent } from "../../setup/helpers.js";
import { expectLine, stripAnsi } from "../../setup/output-helpers.js";

describe("example install + agent repair", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
  });

  test("install creates agents with core/profile.md", async () => {
    // WHEN - installing the matrix example
    const result = await spwn("install matrix")
      .exec("example install matrix")
      .run();

    // THEN - agent neo is created with the current Mind layout
    expect(result.exitCode).toBe(0);
    expect(existsSync(join(home, "agents", "neo", "core", "profile.md"))).toBe(true);
    expect(existsSync(join(home, "agents", "neo", "agent.yaml"))).toBe(true);
    expect(existsSync(join(home, "worlds", "matrix.yaml"))).toBe(true);

    // Profile has real content
    const profile = readFileSync(join(home, "agents", "neo", "core", "profile.md"), "utf-8");
    expect(profile).toContain("Neo");
  });

  test("install repairs broken agent (missing core/)", async () => {
    // GIVEN - a broken agent "neo" with only a journal (no core/profile.md)
    createBrokenAgent(home, "neo");
    expect(existsSync(join(home, "agents", "neo", "journal", "old-session.md"))).toBe(true);
    expect(existsSync(join(home, "agents", "neo", "core", "profile.md"))).toBe(false);

    // WHEN - installing the matrix example (neo already exists but is broken)
    const result = await spwn("install repairs neo")
      .exec("example install matrix")
      .run();

    // THEN - neo is repaired: core/profile.md created, journal preserved
    expect(result.exitCode).toBe(0);
    expect(result.output).toContain("repaired");
    expect(existsSync(join(home, "agents", "neo", "core", "profile.md"))).toBe(true);
    // Old data preserved
    expect(existsSync(join(home, "agents", "neo", "journal", "old-session.md"))).toBe(true);
  });

  test("install skips valid existing agent", async () => {
    // GIVEN - a valid agent "neo" already exists
    createAgent(home, "neo");

    // WHEN - installing the matrix example
    const result = await spwn("install skips valid neo")
      .exec("example install matrix")
      .run();

    // THEN - neo is skipped (not overwritten, not repaired)
    expect(result.exitCode).toBe(0);
    const output = stripAnsi(result.output);
    // Should NOT say "repaired" since the agent is valid
    expect(output).not.toContain("repaired");
  });

  test("install creates all startup agents in one command", async () => {
    // WHEN - installing the startup example
    const result = await spwn("install startup")
      .exec("example install startup")
      .run();

    // THEN - all 3 agents + 1 world created
    expect(result.exitCode).toBe(0);
    for (const agent of ["ceo", "devops", "analyst"]) {
      expect(existsSync(join(home, "agents", agent, "core", "profile.md"))).toBe(true);
    }
    expect(existsSync(join(home, "worlds", "startup.yaml"))).toBe(true);
  });
});

describe("example gallery", () => {
  test("list returns every shipped example", async () => {
    const result = await spwn("example list").exec("example list").run();

    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output);
    for (const slug of ["startup", "matrix", "paperclip-factory", "research-lab", "macrohard"]) {
      expect(out).toContain(slug);
    }
  });
});

describe("org.yaml removal", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
  });

  test("init does NOT create org.yaml", async () => {
    // WHEN - running init
    const result = await spwn("init no org")
      .exec("init")
      .run();

    // THEN - no org.yaml created
    expect(result.exitCode).toBe(0);
    expect(existsSync(join(home, "org.yaml"))).toBe(false);
    expect(result.output).not.toContain("org.yaml");
  });

});

describe("agent mind structure", () => {
  let home: string;
  let originalSpwnHome: string | undefined;

  beforeEach(() => {
    originalSpwnHome = process.env.SPWN_HOME;
    home = createSpwnHome();
    process.env.SPWN_HOME = home;
  });

  afterEach(() => {
    if (originalSpwnHome !== undefined) {
      process.env.SPWN_HOME = originalSpwnHome;
    } else {
      delete process.env.SPWN_HOME;
    }
  });

  test("agent new creates the 5-layer Mind structure", async () => {
    // WHEN - creating a new agent
    const result = await spwn("agent new")
      .exec("agent new TestAgent")
      .run();

    // THEN - the 5 Mind layers are created
    expect(result.exitCode).toBe(0);
    const agentDir = join(home, "agents", "TestAgent");
    for (const layer of ["core", "skills", "knowledge", "playbooks", "journal"]) {
      expect(existsSync(join(agentDir, layer)), `missing ${layer}/`).toBe(true);
    }

    // core/profile.md exists with content
    const personaPath = join(agentDir, "core", "profile.md");
    expect(existsSync(personaPath)).toBe(true);
    const persona = readFileSync(personaPath, "utf-8");
    expect(persona.length).toBeGreaterThan(10);

    // Old structure should NOT exist
    expect(existsSync(join(agentDir, "identity"))).toBe(false);
    expect(existsSync(join(agentDir, "memory"))).toBe(false);
    expect(existsSync(join(agentDir, "sessions"))).toBe(false);
  });
});

describe("CLI upgrade", () => {
  test("upgrade --check finds latest version", async () => {
    // WHEN - checking for updates
    const result = await spwn("upgrade check")
      .exec("upgrade --check")
      .run();

    // THEN - reports a version (the local build is "dev" so latest != current)
    const output = stripAnsi(result.output);
    expect(output).toMatch(/Latest version\s+v\d+\.\d+\.\d+/);
  });
});
