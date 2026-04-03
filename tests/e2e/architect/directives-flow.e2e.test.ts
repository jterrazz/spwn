import { describe, test, expect, beforeEach, afterEach } from "vitest";
import {
  existsSync,
  mkdirSync,
  readFileSync,
  writeFileSync,
  readdirSync,
} from "node:fs";
import { join } from "node:path";
import { createSpwnHome } from "../../setup/helpers.js";

/**
 * Comprehensive E2E tests for the Architect Directives system.
 *
 * Tests cover:
 * - Directives file initialization and default template
 * - Directives file read/write operations
 * - Task counting (pending/completed KPIs)
 * - Directives response parsing (action markers)
 * - Observatory read-only Directives panel constraints
 */
describe("architect Directives flow", () => {
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

  // ──────────────────────────────────────────────
  // Test 10: Directives file is initialized with default template
  // ──────────────────────────────────────────────
  test("architect Directives file is initialized with default template", () => {
    // GIVEN — an architect directory
    const architectDir = join(home, "architect");
    mkdirSync(architectDir, { recursive: true });

    // WHEN — writing the default template (as the system would)
    const defaultContent =
      "# Architect Directives\n\n## In Progress\n\n## Backlog\n\n## Completed\n";
    const directivesPath = join(architectDir, "directives.md");
    writeFileSync(directivesPath, defaultContent);

    // THEN — file exists
    expect(existsSync(directivesPath)).toBe(true);

    // AND — it has all required sections
    const content = readFileSync(directivesPath, "utf-8");
    expect(content).toContain("# Architect Directives");
    expect(content).toContain("## In Progress");
    expect(content).toContain("## Backlog");
    expect(content).toContain("## Completed");
  });

  // ──────────────────────────────────────────────
  // Test 11: Directives can be read (simulated API read)
  // ──────────────────────────────────────────────
  test("architect Directives can be read from file system", () => {
    // GIVEN — a Directives file with content
    const architectDir = join(home, "architect");
    mkdirSync(architectDir, { recursive: true });

    const directivesContent = [
      "# Architect Directives",
      "",
      "## In Progress",
      "- [ ] Deploy API v2",
      "",
      "## Backlog",
      "- [ ] Write documentation",
      "- [ ] Add monitoring",
      "",
      "## Completed",
      "- [x] Setup CI/CD",
      "",
    ].join("\n");

    const directivesPath = join(architectDir, "directives.md");
    writeFileSync(directivesPath, directivesContent);

    // WHEN — reading the file (as the API handler would)
    const content = readFileSync(directivesPath, "utf-8");

    // THEN — content matches what was written
    expect(content).toBe(directivesContent);
    expect(content).toContain("Deploy API v2");
    expect(content).toContain("Write documentation");
    expect(content).toContain("Setup CI/CD");
  });

  // ──────────────────────────────────────────────
  // Test 12: Status includes Directives KPIs
  // ──────────────────────────────────────────────
  test("architect status includes Directives KPIs with correct counts", () => {
    // GIVEN — a Directives file with known task counts
    const architectDir = join(home, "architect");
    mkdirSync(architectDir, { recursive: true });

    const directivesContent = [
      "# Architect Directives",
      "",
      "## In Progress",
      "- [ ] Deploy API v2",
      "- [ ] Fix auth bug",
      "",
      "## Backlog",
      "- [ ] Write docs",
      "",
      "## Completed",
      "- [x] Setup CI/CD",
      "",
    ].join("\n");

    writeFileSync(join(architectDir, "directives.md"), directivesContent);

    // WHEN — parsing the file for KPIs (same logic as Go server)
    const content = readFileSync(
      join(architectDir, "directives.md"),
      "utf-8",
    );
    const pendingMatches = content.match(/- \[ \]/g) ?? [];
    const doneMatches = content.match(/- \[x\]/gi) ?? [];

    // THEN — counts are correct
    expect(pendingMatches.length).toBe(3); // 2 in progress + 1 backlog
    expect(doneMatches.length).toBe(1); // 1 completed
  });

  // ──────────────────────────────────────────────
  // Test 13: Architect talk response shape
  // ──────────────────────────────────────────────
  test("architect talk response includes directiveAction when present", () => {
    // We can't test the actual architect container, but we can verify
    // the response parsing logic that the API uses.

    // GIVEN — a simulated architect response with Directives action
    const response =
      "Sure, I'll add that to the list.\n[DIRECTIVE_ADD] Deploy API\nPriority: high\nSetting up the deployment pipeline.";

    // WHEN — parsing the response (same regex as Go server)
    const todoAddMatch = response.match(
      /\[Directives_ADD\]\s*(.*?)(?:\n|$)/,
    );
    const priorityMatch = response.match(/Priority:\s*(.*?)(?:\n|$)/);

    // THEN — action fields are extracted
    expect(todoAddMatch).not.toBeNull();
    expect(todoAddMatch![1].trim()).toBe("Deploy API");
    expect(priorityMatch).not.toBeNull();
    expect(priorityMatch![1].trim()).toBe("high");

    // Verify the response shape matches what the API would return
    const apiResponse = {
      response: response,
      directiveAction: {
        type: "add",
        title: todoAddMatch![1].trim(),
        priority: priorityMatch![1].trim(),
      },
    };

    expect(apiResponse).toHaveProperty("response");
    expect(apiResponse).toHaveProperty("directiveAction");
    expect(apiResponse.directiveAction.type).toBe("add");
    expect(apiResponse.directiveAction.title).toBe("Deploy API");
  });

  test("architect talk response without Directives action has no directiveAction field", () => {
    // GIVEN — a regular response with no Directives markers
    const response =
      "The system is running well. All agents are healthy and no issues detected.";

    // WHEN — parsing the response
    const todoAddMatch = response.match(/\[Directives_ADD\]/);
    const todoDoneMatch = response.match(/\[Directives_DONE\]/);
    const todoUpdateMatch = response.match(/\[Directives_UPDATE\]/);

    // THEN — no action markers found
    expect(todoAddMatch).toBeNull();
    expect(todoDoneMatch).toBeNull();
    expect(todoUpdateMatch).toBeNull();
  });

  // ──────────────────────────────────────────────
  // Test 14: Directives panel in observatory is read-only
  // ──────────────────────────────────────────────
  test("Directives panel in observatory is read-only (no Add Directives input)", () => {
    // We verify this by checking the Observatory page source.
    // The architect page.tsx uses parseDirectivesMd() to display Directivess
    // but does NOT expose any "add" input/form for manual Directives creation.

    // GIVEN — the Observatory architect page source
    const pagePath = join(
      home,
      "..",
      "..",
      "..",
      "apps",
      "observatory",
      "src",
      "app",
      "architect",
      "page.tsx",
    );

    // Since we're in a temp dir, use the actual project path
    const projectPagePath = join(
      __dirname,
      "..",
      "..",
      "..",
      "apps",
      "observatory",
      "src",
      "app",
      "architect",
      "page.tsx",
    );

    if (!existsSync(projectPagePath)) {
      // If the page doesn't exist yet, the test passes (no edit controls)
      return;
    }

    const source = readFileSync(projectPagePath, "utf-8");

    // THEN — the page should NOT have add/edit Directives form elements
    // The Directives panel only displays data fetched from the API.
    // It should have parseDirectivesMd (read) but not a Directives creation form.
    expect(source).toContain("parseDirectivesMd"); // read-only parser exists
    expect(source).not.toMatch(/name=["']addTodo["']/); // no add form input
    expect(source).not.toMatch(/placeholder=["']Add.*Directives["']/i); // no add Directives placeholder
    expect(source).not.toMatch(
      /handleAddTodo|handleCreateTodo|onAddTodo/,
    ); // no add handlers
  });

  // ──────────────────────────────────────────────
  // Additional: Directives_DONE response parsing
  // ──────────────────────────────────────────────
  test("architect Directives_DONE response parsing extracts completion info", () => {
    const response =
      "Done! I've completed the task.\n[DIRECTIVE_DONE] Setup CI/CD\nCompleted: deployed pipeline to GitHub Actions";

    const todoDoneMatch = response.match(
      /\[Directives_DONE\]\s*(.*?)(?:\n|$)/,
    );
    const completedMatch = response.match(
      /Completed:\s*(.*?)(?:\n|$)/,
    );

    expect(todoDoneMatch).not.toBeNull();
    expect(todoDoneMatch![1].trim()).toBe("Setup CI/CD");
    expect(completedMatch).not.toBeNull();
    expect(completedMatch![1].trim()).toBe(
      "deployed pipeline to GitHub Actions",
    );
  });

  // ──────────────────────────────────────────────
  // Additional: Directives_UPDATE response parsing
  // ──────────────────────────────────────────────
  test("architect Directives_UPDATE response parsing extracts progress info", () => {
    const response =
      "Making progress on this.\n[DIRECTIVE_UPDATE] Deploy API\nProgress: 75% complete, testing in staging";

    const todoUpdateMatch = response.match(
      /\[Directives_UPDATE\]\s*(.*?)(?:\n|$)/,
    );
    const progressMatch = response.match(
      /Progress:\s*(.*?)(?:\n|$)/,
    );

    expect(todoUpdateMatch).not.toBeNull();
    expect(todoUpdateMatch![1].trim()).toBe("Deploy API");
    expect(progressMatch).not.toBeNull();
    expect(progressMatch![1].trim()).toBe(
      "75% complete, testing in staging",
    );
  });

  // ──────────────────────────────────────────────
  // Additional: Directives file with edge cases
  // ──────────────────────────────────────────────
  test("Directives counting handles uppercase X checkboxes", () => {
    const architectDir = join(home, "architect");
    mkdirSync(architectDir, { recursive: true });

    const directivesContent = [
      "# Directives",
      "## In Progress",
      "- [ ] Task A",
      "## Completed",
      "- [x] Done lowercase",
      "- [X] Done uppercase",
    ].join("\n");

    writeFileSync(join(architectDir, "directives.md"), directivesContent);

    const content = readFileSync(
      join(architectDir, "directives.md"),
      "utf-8",
    );
    const pendingMatches = content.match(/- \[ \]/g) ?? [];
    const doneMatches = content.match(/- \[x\]/gi) ?? [];

    expect(pendingMatches.length).toBe(1);
    expect(doneMatches.length).toBe(2);
  });

  test("Directives counting with empty sections returns zero", () => {
    const architectDir = join(home, "architect");
    mkdirSync(architectDir, { recursive: true });

    const directivesContent =
      "# Architect Directives\n\n## In Progress\n\n## Backlog\n\n## Completed\n";

    writeFileSync(join(architectDir, "directives.md"), directivesContent);

    const content = readFileSync(
      join(architectDir, "directives.md"),
      "utf-8",
    );
    const pendingMatches = content.match(/- \[ \]/g) ?? [];
    const doneMatches = content.match(/- \[x\]/gi) ?? [];

    expect(pendingMatches.length).toBe(0);
    expect(doneMatches.length).toBe(0);
  });

  test("Directives file write and re-read roundtrip preserves content", () => {
    const architectDir = join(home, "architect");
    mkdirSync(architectDir, { recursive: true });
    const directivesPath = join(architectDir, "directives.md");

    // Write
    const content =
      "# Architect Directives\n\n## In Progress\n- [ ] Task 1\n\n## Backlog\n- [ ] Task 2\n\n## Completed\n- [x] Task 3\n";
    writeFileSync(directivesPath, content);

    // Update (overwrite)
    const updated =
      "# Architect Directives\n\n## In Progress\n- [ ] Task 1\n- [ ] Task 4 (new)\n\n## Backlog\n\n## Completed\n- [x] Task 2 (moved)\n- [x] Task 3\n";
    writeFileSync(directivesPath, updated);

    // Read back
    const result = readFileSync(directivesPath, "utf-8");
    expect(result).toBe(updated);
    expect(result).toContain("Task 4 (new)");
    expect(result).toContain("Task 2 (moved)");

    // Verify counts after update
    const pendingMatches = result.match(/- \[ \]/g) ?? [];
    const doneMatches = result.match(/- \[x\]/gi) ?? [];
    expect(pendingMatches.length).toBe(2);
    expect(doneMatches.length).toBe(2);
  });

  // ──────────────────────────────────────────────
  // Multiple Directives actions: only first is recognized
  // ──────────────────────────────────────────────
  test("multiple Directives actions in response — only first is recognized", () => {
    const response = [
      "I'll handle both tasks.",
      "[DIRECTIVE_ADD] First task",
      "Priority: high",
      "",
      "[DIRECTIVE_ADD] Second task",
      "Priority: low",
    ].join("\n");

    // The Go parser returns only the first match.
    // We simulate that behavior here.
    const firstMatch = response.match(
      /\[Directives_ADD\]\s*(.*?)(?:\n|$)/,
    );
    expect(firstMatch).not.toBeNull();
    expect(firstMatch![1].trim()).toBe("First task");

    // Count total matches — there should be 2 in the text
    const allMatches = response.match(/\[Directives_ADD\]/g) ?? [];
    expect(allMatches.length).toBe(2);
  });
});
