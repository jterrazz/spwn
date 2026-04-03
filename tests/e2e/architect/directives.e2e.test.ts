import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { createSpwnHome } from "../../setup/helpers.js";

describe("architect Directives", () => {
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

  test("architect Directives directory can be created", () => {
    // GIVEN — a SPWN_HOME
    const architectDir = join(home, "architect");
    mkdirSync(architectDir, { recursive: true });

    // WHEN — writing a directives.md file
    const directivesPath = join(architectDir, "directives.md");
    const content = [
      "# Architect Directives",
      "",
      "## In Progress",
      "- [ ] Set up initial agent fleet",
      "",
      "## Backlog",
      "- [ ] Configure monitoring",
      "- [ ] Add error handling",
      "",
      "## Completed",
      "- [x] Initialize project structure",
      "",
    ].join("\n");
    writeFileSync(directivesPath, content);

    // THEN — file exists and is readable
    expect(existsSync(directivesPath)).toBe(true);
    const read = readFileSync(directivesPath, "utf-8");
    expect(read).toContain("## In Progress");
    expect(read).toContain("## Backlog");
    expect(read).toContain("## Completed");
  });

  test("architect Directives default template has expected sections", () => {
    // GIVEN — a fresh architect directory
    const architectDir = join(home, "architect");
    mkdirSync(architectDir, { recursive: true });

    // WHEN — writing the default template
    const defaultContent =
      "# Architect Directives\n\n## In Progress\n\n## Backlog\n\n## Completed\n";
    const directivesPath = join(architectDir, "directives.md");
    writeFileSync(directivesPath, defaultContent);

    // THEN — template has all required sections
    const content = readFileSync(directivesPath, "utf-8");
    expect(content).toContain("# Architect Directives");
    expect(content).toContain("## In Progress");
    expect(content).toContain("## Backlog");
    expect(content).toContain("## Completed");
  });

  test("architect Directives supports checkbox parsing", () => {
    // GIVEN — a Directives file with checkboxes
    const architectDir = join(home, "architect");
    mkdirSync(architectDir, { recursive: true });
    const directivesPath = join(architectDir, "directives.md");
    writeFileSync(
      directivesPath,
      [
        "# Directives",
        "## In Progress",
        "- [ ] Task A",
        "- [ ] Task B",
        "## Backlog",
        "- [ ] Task C",
        "## Completed",
        "- [x] Task D",
        "- [x] Task E",
      ].join("\n"),
    );

    // WHEN — reading and parsing
    const content = readFileSync(directivesPath, "utf-8");
    const pendingMatches = content.match(/- \[ \]/g) ?? [];
    const doneMatches = content.match(/- \[x\]/g) ?? [];

    // THEN — counts are correct
    expect(pendingMatches.length).toBe(3);
    expect(doneMatches.length).toBe(2);
  });
});
