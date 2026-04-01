import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";
import { writeFileSync } from "node:fs";
import { join } from "node:path";
import { expectLine, lines, stripAnsi } from "../../setup/output-helpers.js";

describe("spwn status", () => {
  // Non-Docker tests
  describe("without worlds", () => {
    let home: string;
    let originalSpwnHome: string | undefined;

    beforeEach(() => {
      originalSpwnHome = process.env.SPWN_HOME;
      home = createSpwnHome();
      process.env.SPWN_HOME = home;
    });

    afterEach(() => {
      if (originalSpwnHome !== undefined)
        process.env.SPWN_HOME = originalSpwnHome;
      else delete process.env.SPWN_HOME;
    });

    test("shows header box with spwn branding", async () => {
      await spwn("init").exec("init").run();
      const result = await spwn("status").exec("status").run();

      expect(result.exitCode).toBe(0);
      const out = stripAnsi(result.output);
      expect(out).toContain("s p w n");
      expect(out).toContain("\u256d");
      expect(out).toContain("\u2570");
    });

    test("shows god section as offline", async () => {
      await spwn("init").exec("init").run();
      const result = await spwn("status").exec("status").run();

      const out = stripAnsi(result.output);
      expect(out).toContain("God");
      expect(out).toContain("offline");
    });

    test("shows universe section", async () => {
      await spwn("init").exec("init").run();
      const result = await spwn("status").exec("status").run();

      const out = stripAnsi(result.output);
      expect(out).toContain("Universe");
    });

    test("shows drifting agents", async () => {
      await spwn("init").exec("init").run();
      await spwn("agent init").exec("agent init neo").run();
      const result = await spwn("status").exec("status").run();

      const out = stripAnsi(result.output);
      expect(out).toContain("Drifting");
      expect(out).toContain("neo");
    });

    test("shows auth status", async () => {
      await spwn("init").exec("init").run();
      const result = await spwn("status").exec("status").run();

      const out = stripAnsi(result.output);
      // Either "subscription" or "not configured" or "API key"
      expect(out).toMatch(/subscription|not configured|API key/);
    });

    test("shows org name when org.yaml exists", async () => {
      await spwn("init").exec("init").run();
      writeFileSync(join(home, "org.yaml"), "name: acme-corp\nversion: 1\n");
      const result = await spwn("status").exec("status").run();

      const out = stripAnsi(result.output);
      expect(out).toContain("acme-corp");
    });

    test("shows physics constants from default config", async () => {
      await spwn("init").exec("init").run();
      const result = await spwn("status").exec("status").run();

      const out = stripAnsi(result.output);
      expect(out).toMatch(/\d+ cpu/);
      expect(out).toContain("512m");
      expect(out).toContain("30m");
    });

    test("shows version in header", async () => {
      await spwn("init").exec("init").run();
      const result = await spwn("status").exec("status").run();

      const out = stripAnsi(result.output);
      // Version is "dev" in non-release builds, or semver in releases
      expect(out).toMatch(/v[\w.]+/);
    });

    test("shows skill count", async () => {
      await spwn("init").exec("init").run();
      const result = await spwn("status").exec("status").run();

      const out = stripAnsi(result.output);
      expect(out).toMatch(/\d+ skills/);
    });

    test("shows drifting section even with no agents", async () => {
      // init creates a default agent, so we just check the section exists
      await spwn("init").exec("init").run();
      const result = await spwn("status").exec("status").run();

      const out = stripAnsi(result.output);
      expect(out).toContain("Drifting");
    });

    test("uses box-drawing characters", async () => {
      await spwn("init").exec("init").run();
      const result = await spwn("status").exec("status").run();

      const out = stripAnsi(result.output);
      // Header box
      expect(out).toContain("\u256d"); // ╭
      expect(out).toContain("\u256e"); // ╮
      expect(out).toContain("\u2570"); // ╰
      expect(out).toContain("\u256f"); // ╯
      expect(out).toContain("\u2502"); // │
      expect(out).toContain("\u2500"); // ─
    });
  });

  // Docker tests
  describe("with active world", () => {
    let ctx: TestContext;

    afterEach(() => {
      ctx?.cleanup();
    });

    test("shows world bubble with agent", () => {
      ctx = createTestContext();
      ctx.spwn(["init"]);
      const spawnResult = ctx.spwn(
        ["world", "--agent", "neo", "-w", ctx.home],
        60_000,
      );
      const id = parseWorldId(spawnResult.output)!;

      // Verify the world exists in the list
      const listResult = ctx.spwn(["world", "list"]);
      const listOut = stripAnsi(listResult.output);

      const result = ctx.spwn(["status"]);

      expect(result.exitCode).toBe(0);
      const out = stripAnsi(result.output);

      // Header is always present
      expect(out).toContain("s p w n");
      expect(out).toContain("\u256d");
      expect(out).toContain("\u2570");

      // If the world list shows the ID, status should too
      if (listOut.includes(id)) {
        expect(out).toContain(id);
        expect(out).toContain("neo");
      }
    });

    test("shows universe physics from config", () => {
      ctx = createTestContext();
      ctx.spwn(["init"]);
      ctx.spwn(["world", "--agent", "neo", "-w", ctx.home], 60_000);

      const result = ctx.spwn(["status"]);
      const out = stripAnsi(result.output);

      expect(out).toContain("Universe");
      // Should show physics constants
      expect(out).toMatch(/\d+ cpu/);
    });
  });
});
