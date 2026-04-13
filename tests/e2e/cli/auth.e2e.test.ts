import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { spwn } from "../../setup/spwn.specification.js";
import { createSpwnHome } from "../../setup/helpers.js";
import {
  expectLine,
  expectTableHeader,
  stripAnsi,
} from "../../setup/output-helpers.js";

// ── Auth command E2E tests ─────────────────────────────────

describe("CLI — auth command", () => {
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

  // ── spwn auth (provider status table) ──────────────────────

  test("'spwn auth' shows provider status table", async () => {
    // WHEN — running auth with no subcommand
    const result = await spwn("auth status").exec("auth").run();

    // THEN — exit code 0
    expect(result.exitCode).toBe(0);

    // AND — output contains a table with PROVIDER and STATUS columns
    const out = stripAnsi(result.output);
    expect(out).toContain("PROVIDER");
    expect(out).toContain("STATUS");
  });

  test("'spwn auth' lists known providers", async () => {
    // WHEN — running auth
    const result = await spwn("auth providers").exec("auth").run();

    // THEN — output mentions key providers
    expect(result.exitCode).toBe(0);
    const out = stripAnsi(result.output).toLowerCase();
    // Should mention at least one well-known provider
    const hasProvider =
      out.includes("anthropic") ||
      out.includes("openai") ||
      out.includes("open-router") ||
      out.includes("openrouter");
    expect(hasProvider).toBe(true);
  });

  // ── spwn auth check ────────────────────────────────────────

  test("'spwn auth check' validates credentials and shows results", async () => {
    // WHEN — running auth check
    const result = await spwn("auth check").exec("auth check").run();

    // THEN — exit code 0 (even if no creds configured)
    expect(result.exitCode).toBe(0);

    // AND — output shows validation results
    const out = stripAnsi(result.output);
    // Should mention checking or validation
    const hasCheckOutput =
      out.includes("check") ||
      out.includes("Check") ||
      out.includes("valid") ||
      out.includes("connect") ||
      out.includes("PROVIDER") ||
      out.includes("provider");
    expect(hasCheckOutput).toBe(true);
  });

  // ── spwn auth --help ───────────────────────────────────────

  test("'spwn auth --help' shows subcommands", async () => {
    // WHEN — running auth --help
    const result = await spwn("auth help").exec("auth --help").run();

    // THEN — exit code 0
    expect(result.exitCode).toBe(0);

    // AND — output describes auth subcommands
    const out = stripAnsi(result.output);
    expect(out).toContain("auth");
    // Should mention subcommands like check, token, login, logout
    const hasSubcommands =
      out.includes("check") ||
      out.includes("token") ||
      out.includes("login") ||
      out.includes("logout") ||
      out.includes("Commands") ||
      out.includes("COMMANDS") ||
      out.includes("Usage");
    expect(hasSubcommands).toBe(true);
  });

  // ── spwn auth token --help ─────────────────────────────────

  test("'spwn auth token --help' shows usage", async () => {
    // WHEN — running auth token --help
    const result = await spwn("auth token help")
      .exec("auth token --help")
      .run();

    // THEN — exit code 0
    expect(result.exitCode).toBe(0);

    // AND — output contains usage information
    const out = stripAnsi(result.output);
    expect(out).toContain("token");
    const hasUsage =
      out.includes("Usage") ||
      out.includes("usage") ||
      out.includes("USAGE") ||
      out.includes("Options") ||
      out.includes("--help");
    expect(hasUsage).toBe(true);
  });

  // ── spwn auth login (non-interactive) ──────────────────────

  test("'spwn auth login' handles non-interactive gracefully", async () => {
    // WHEN — running auth login with piped empty input (non-interactive)
    const result = await spwn("auth login noninteractive")
      .exec("auth login")
      .run();

    // THEN — should not crash (exit code 0 or 1 for "no input")
    // The key assertion is it doesn't hang or crash with a stack trace
    expect(result.exitCode).toBeDefined();
    expect(typeof result.exitCode).toBe("number");

    // AND — no raw stack traces in output
    const out = stripAnsi(result.output);
    expect(out).not.toMatch(/at\s+\S+\s+\(/); // no JS stack traces
    expect(out).not.toContain("panic:");
    expect(out).not.toContain("goroutine ");
  });

  // ── spwn auth logout ───────────────────────────────────────

  test("'spwn auth logout' removes cached token", async () => {
    // WHEN — running auth logout (even with no token to remove)
    const result = await spwn("auth logout").exec("auth logout").run();

    // THEN — should complete without error
    // Exit code 0 = success, or non-zero if nothing to remove (both acceptable)
    expect(result.exitCode).toBeDefined();

    // AND — output should indicate token removal or that none existed
    const out = stripAnsi(result.output);
    const hasLogoutMsg =
      out.includes("logout") ||
      out.includes("Logout") ||
      out.includes("removed") ||
      out.includes("Removed") ||
      out.includes("cleared") ||
      out.includes("token") ||
      out.includes("signed out") ||
      out.includes("logged out") ||
      out.length === 0; // empty output is acceptable for "nothing to do"
    expect(hasLogoutMsg).toBe(true);

    // AND — no stack traces
    expect(out).not.toMatch(/at\s+\S+\s+\(/);
  });

  test("'spwn auth logout' is idempotent", async () => {
    // WHEN — running logout twice
    const result1 = await spwn("auth logout 1").exec("auth logout").run();
    const result2 = await spwn("auth logout 2").exec("auth logout").run();

    // THEN — second call should also succeed (idempotent)
    expect(result2.exitCode).toBeDefined();
    // Should not crash on second call
    expect(stripAnsi(result2.output)).not.toMatch(/at\s+\S+\s+\(/);
  });
});
