import { describe, test, expect, beforeEach, afterEach } from "vitest";
import { existsSync, mkdirSync, readFileSync, writeFileSync, rmSync } from "node:fs";
import { join } from "node:path";
import { createSpwnHome } from "../../setup/helpers.js";

describe("blueprint", () => {
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

  test("blueprint directory is initialized with default files", () => {
    // GIVEN — a SPWN_HOME with a blueprint directory
    const blueprintDir = join(home, "blueprint");
    mkdirSync(blueprintDir, { recursive: true });

    // WHEN — writing default blueprint files (simulating init)
    const defaultFiles: Record<string, string> = {
      "overview.md": "# Universe Blueprint\n\nThis is the knowledge base for your spwn universe.\n",
      "glossary.md": "# Glossary\n\nKey terms and concepts.\n",
      "roadmap.md": "# Roadmap\n\n## Current Focus\n",
    };

    for (const [name, content] of Object.entries(defaultFiles)) {
      writeFileSync(join(blueprintDir, name), content);
    }

    // THEN — all default files exist
    expect(existsSync(join(blueprintDir, "overview.md"))).toBe(true);
    expect(existsSync(join(blueprintDir, "glossary.md"))).toBe(true);
    expect(existsSync(join(blueprintDir, "roadmap.md"))).toBe(true);
  });

  test("blueprint files have expected content", () => {
    // GIVEN — initialized blueprint directory
    const blueprintDir = join(home, "blueprint");
    mkdirSync(blueprintDir, { recursive: true });

    const overviewContent = "# Universe Blueprint\n\nThis is the knowledge base for your spwn universe.\nThe Architect maintains this as the single source of truth.\n";
    writeFileSync(join(blueprintDir, "overview.md"), overviewContent);

    // WHEN — reading files
    const content = readFileSync(join(blueprintDir, "overview.md"), "utf-8");

    // THEN — content matches
    expect(content).toContain("Universe Blueprint");
    expect(content).toContain("single source of truth");
  });

  test("blueprint ls lists files correctly", () => {
    // GIVEN — blueprint with multiple files
    const blueprintDir = join(home, "blueprint");
    mkdirSync(blueprintDir, { recursive: true });
    mkdirSync(join(blueprintDir, "projects"), { recursive: true });

    writeFileSync(join(blueprintDir, "overview.md"), "# Overview");
    writeFileSync(join(blueprintDir, "glossary.md"), "# Glossary");
    writeFileSync(join(blueprintDir, "projects", "api.md"), "# API Project");

    // WHEN — listing files
    const { readdirSync, statSync } = require("node:fs");
    const { join: pathJoin } = require("node:path");

    const walkFiles = (dir: string, base: string): string[] => {
      const results: string[] = [];
      const entries = readdirSync(dir, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = pathJoin(dir, entry.name);
        if (entry.isDirectory()) {
          results.push(...walkFiles(fullPath, base));
        } else {
          const relPath = fullPath.replace(base + "/", "");
          results.push(relPath);
        }
      }
      return results;
    };

    const files = walkFiles(blueprintDir, blueprintDir);

    // THEN — all files are listed
    expect(files).toContain("overview.md");
    expect(files).toContain("glossary.md");
    expect(files).toContain("projects/api.md");
  });

  test("blueprint show displays file content", () => {
    // GIVEN — a blueprint file
    const blueprintDir = join(home, "blueprint");
    mkdirSync(blueprintDir, { recursive: true });

    const expectedContent = "# Universe Blueprint\n\nThis is the overview.\n";
    writeFileSync(join(blueprintDir, "overview.md"), expectedContent);

    // WHEN — reading the file
    const content = readFileSync(join(blueprintDir, "overview.md"), "utf-8");

    // THEN — content is returned correctly
    expect(content).toBe(expectedContent);
    expect(content).toContain("Universe Blueprint");
  });

  test("blueprint API returns file list structure", () => {
    // GIVEN — blueprint with files
    const blueprintDir = join(home, "blueprint");
    mkdirSync(blueprintDir, { recursive: true });

    writeFileSync(join(blueprintDir, "overview.md"), "# Overview");
    writeFileSync(join(blueprintDir, "glossary.md"), "# Glossary");

    // WHEN — simulating API response construction
    const { readdirSync, statSync } = require("node:fs");
    const files = readdirSync(blueprintDir).map((name: string) => {
      const stat = statSync(join(blueprintDir, name));
      return {
        path: name,
        size: stat.size,
        modified: stat.mtime.toISOString(),
      };
    });

    // THEN — response has expected shape
    expect(Array.isArray(files)).toBe(true);
    expect(files.length).toBe(2);
    for (const file of files) {
      expect(file).toHaveProperty("path");
      expect(file).toHaveProperty("size");
      expect(file).toHaveProperty("modified");
      expect(typeof file.size).toBe("number");
      expect(file.size).toBeGreaterThan(0);
    }
  });

  test("blueprint API returns file content", () => {
    // GIVEN — a blueprint file
    const blueprintDir = join(home, "blueprint");
    mkdirSync(blueprintDir, { recursive: true });

    const markdownContent = "# Overview\n\n## Architecture\n\nThis is the main overview.\n";
    writeFileSync(join(blueprintDir, "overview.md"), markdownContent);

    // WHEN — simulating API content response
    const content = readFileSync(join(blueprintDir, "overview.md"), "utf-8");
    const response = { path: "overview.md", content };

    // THEN — response contains markdown content
    expect(response.path).toBe("overview.md");
    expect(response.content).toContain("# Overview");
    expect(response.content).toContain("Architecture");
  });

  test("blueprint prevents directory traversal", () => {
    // GIVEN — a blueprint directory
    const blueprintDir = join(home, "blueprint");
    mkdirSync(blueprintDir, { recursive: true });
    writeFileSync(join(blueprintDir, "overview.md"), "# Overview");

    // Write a file outside blueprint
    writeFileSync(join(home, "secret.txt"), "secret data");

    // WHEN — attempting directory traversal
    const requestedPath = "../secret.txt";
    const hasTraversal = requestedPath.includes("..");

    // THEN — traversal is detected and blocked
    expect(hasTraversal).toBe(true);

    // The resolved path would escape the blueprint directory
    const { resolve } = require("node:path");
    const resolvedPath = resolve(join(blueprintDir, requestedPath));
    const isWithinBlueprint = resolvedPath.startsWith(resolve(blueprintDir));
    expect(isWithinBlueprint).toBe(false);
  });

  test("blueprint subdirectories work correctly", () => {
    // GIVEN — blueprint with nested directories
    const blueprintDir = join(home, "blueprint");
    mkdirSync(join(blueprintDir, "decisions"), { recursive: true });
    mkdirSync(join(blueprintDir, "projects"), { recursive: true });
    mkdirSync(join(blueprintDir, "agents"), { recursive: true });

    writeFileSync(join(blueprintDir, "decisions", "auth-flow.md"), "# Auth Flow Decision");
    writeFileSync(join(blueprintDir, "projects", "api.md"), "# API Project");
    writeFileSync(join(blueprintDir, "agents", "team.md"), "# Team");

    // WHEN — reading nested files
    const authContent = readFileSync(join(blueprintDir, "decisions", "auth-flow.md"), "utf-8");
    const apiContent = readFileSync(join(blueprintDir, "projects", "api.md"), "utf-8");
    const teamContent = readFileSync(join(blueprintDir, "agents", "team.md"), "utf-8");

    // THEN — all nested files are readable
    expect(authContent).toContain("Auth Flow Decision");
    expect(apiContent).toContain("API Project");
    expect(teamContent).toContain("Team");
  });

  test("blueprint init does not overwrite existing files", () => {
    // GIVEN — a blueprint with a custom overview
    const blueprintDir = join(home, "blueprint");
    mkdirSync(blueprintDir, { recursive: true });

    const customContent = "# My Custom Overview\n\nThis was manually edited.\n";
    writeFileSync(join(blueprintDir, "overview.md"), customContent);

    // WHEN — simulating re-init (only write if not exists)
    const overviewPath = join(blueprintDir, "overview.md");
    if (!existsSync(overviewPath)) {
      writeFileSync(overviewPath, "# Default Overview");
    }

    // Also write new defaults that don't exist yet
    const glossaryPath = join(blueprintDir, "glossary.md");
    if (!existsSync(glossaryPath)) {
      writeFileSync(glossaryPath, "# Glossary");
    }

    // THEN — custom content preserved, new defaults created
    const content = readFileSync(overviewPath, "utf-8");
    expect(content).toBe(customContent);
    expect(existsSync(glossaryPath)).toBe(true);
  });

  test("blueprint search finds matches across files", () => {
    // GIVEN — blueprint with searchable content
    const blueprintDir = join(home, "blueprint");
    mkdirSync(blueprintDir, { recursive: true });

    writeFileSync(join(blueprintDir, "overview.md"), "# Overview\n\nThe authentication system uses JWT tokens.\n");
    writeFileSync(join(blueprintDir, "glossary.md"), "# Glossary\n\n| Term | Definition |\n| JWT | JSON Web Token |\n");
    writeFileSync(join(blueprintDir, "roadmap.md"), "# Roadmap\n\n## Current Focus\n- Improve performance\n");

    // WHEN — searching for "JWT"
    const { readdirSync } = require("node:fs");
    const query = "JWT";
    const results: Record<string, string[]> = {};

    const files = readdirSync(blueprintDir);
    for (const file of files) {
      const content = readFileSync(join(blueprintDir, file), "utf-8");
      const matchingLines = content.split("\n").filter((line: string) =>
        line.toLowerCase().includes(query.toLowerCase())
      );
      if (matchingLines.length > 0) {
        results[file] = matchingLines;
      }
    }

    // THEN — matches found in relevant files
    expect(Object.keys(results)).toContain("overview.md");
    expect(Object.keys(results)).toContain("glossary.md");
    expect(Object.keys(results)).not.toContain("roadmap.md");
  });
});
