import { describe, test, expect, afterEach } from "vitest";
import { writeFileSync } from "node:fs";
import { join } from "node:path";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { expectLine } from "../../setup/output-helpers.js";

describe("world security — physics enforcement", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("missing element is truly missing — no @spwn/git means no git binary", () => {
    // GIVEN — a config with only @spwn/unix (no @spwn/git)
    ctx = createTestContext();
    ctx.spwn(["init"]);

    // Override the default config to remove @spwn/git
    const configContent = `physics:
  constants:
    cpu: 1
    memory: 512m
    disk: 2g
    timeout: 30m
  laws:
    max-processes: 256
  elements:
    - "@spwn/unix"
`;
    writeFileSync(join(ctx.home, "worlds", "nogit.yaml"), configContent);

    // WHEN — spawning a world with this config
    const result = ctx.spwn(
      ["world", "-c", "nogit", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;
    expect(id).toBeTruthy();

    // THEN — git should NOT be available inside the container
    let gitFound = false;
    try {
      ctx.universe(id).exec("which git");
      gitFound = true;
    } catch {
      // Expected: git not found → non-zero exit
    }
    expect(gitFound).toBe(false);

    // AND — faculties.md should NOT mention git
    const faculties = ctx.universe(id).faculties();
    expect(faculties).not.toMatch(/\bgit\b/);
  });

  test("element pack expansion — @spwn/unix, @spwn/git, @spwn/node all present", () => {
    // GIVEN — a config with multiple element packs
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const configContent = `physics:
  constants:
    cpu: 1
    memory: 512m
    disk: 2g
    timeout: 30m
  laws:
    max-processes: 256
  elements:
    - "@spwn/unix"
    - "@spwn/git"
    - "@spwn/node"
`;
    writeFileSync(join(ctx.home, "worlds", "fullstack.yaml"), configContent);

    // WHEN — spawning a world with all element packs
    const result = ctx.spwn(
      ["world", "-c", "fullstack", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;
    expect(id).toBeTruthy();

    // THEN — bash, git, and node should all be available
    const bashPath = ctx.universe(id).exec("which bash");
    expect(bashPath).toContain("/bash");

    const gitPath = ctx.universe(id).exec("which git");
    expect(gitPath).toContain("/git");

    const nodePath = ctx.universe(id).exec("which node");
    expect(nodePath).toContain("/node");

    // AND — faculties.md should mention all of them
    const faculties = ctx.universe(id).faculties();
    expect(faculties).toMatch(/bash/);
    expect(faculties).toMatch(/git/);
    expect(faculties).toMatch(/node/);
  });

  test("physics constants are documented in physics.md", () => {
    // GIVEN — a config with specific constants
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const configContent = `physics:
  constants:
    cpu: 2
    memory: 1g
    disk: 4g
    timeout: 30m
  laws:
    max-processes: 256
  elements:
    - "@spwn/unix"
    - "@spwn/git"
`;
    writeFileSync(join(ctx.home, "worlds", "custom.yaml"), configContent);

    // WHEN — spawning a world with these physics
    const result = ctx.spwn(
      ["world", "-c", "custom", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;
    expect(id).toBeTruthy();

    // THEN — physics.md should document the constants
    const physics = ctx.universe(id).physics();
    expect(physics).toMatch(/2/); // CPU count
    expect(physics).toMatch(/1[gG]/i); // Memory
    expect(physics).toMatch(/30m/); // Timeout
  });

  test("network isolation is enforced — container has network=none", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;

    // THEN — Docker network mode is "none"
    const inspectData = ctx.universe(id).inspect();
    expect(inspectData.HostConfig?.NetworkMode).toBe("none");
  });

  test("pids limit is enforced from config", () => {
    // GIVEN — a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);
    const result = ctx.spwn(
      ["world", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    expect(result.exitCode).toBe(0);
    const id = parseWorldId(result.output)!;

    // THEN — pids limit is set (not unlimited)
    const inspectData = ctx.universe(id).inspect();
    const pidsLimit = inspectData.HostConfig?.PidsLimit ?? 0;
    expect(pidsLimit).toBeGreaterThan(0);
  });
});
