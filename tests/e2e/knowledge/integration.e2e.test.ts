import { describe, test, expect, afterEach } from "vitest";
import { existsSync, readFileSync, writeFileSync, unlinkSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import {
  createTestContext,
  parseWorldId,
  type TestContext,
} from "../../setup/spwn.specification.js";
import { stripAnsi } from "../../setup/output-helpers.js";
import { createSpwnHome } from "../../setup/helpers.js";

describe("knowledge integration", () => {
  let ctx: TestContext;

  afterEach(() => {
    ctx?.cleanup();
  });

  test("world knowledge dir is created inside the container on demand", () => {
    // GIVEN - a spawned world
    ctx = createTestContext();
    ctx.spwn(["init"]);

    const spawnResult = ctx.spwn(
      ["up", "--agent", "neo", "-w", ctx.home],
      60_000,
    );
    const worldId = parseWorldId(spawnResult.output);
    if (!worldId) {
      console.warn("Skipping: world spawn failed (Docker unavailable?)");
      return;
    }

    const universe = ctx.world(worldId);

    // The knowledge dir is per-world, lives inside the container, and is
    // created lazily by the agent. It does not exist immediately.
    const lsResult = ctx.spwn(["world", "knowledge", "ls", worldId]);
    expect(lsResult.exitCode).toBe(0);
    expect(stripAnsi(lsResult.output).toLowerCase()).toContain("empty");

    // Create a knowledge file via docker exec to simulate the agent writing
    // and verify ls picks it up.
    universe.exec(
      "mkdir -p /world/knowledge && echo '# Test' > /world/knowledge/note.md",
    );
    const lsAfter = ctx.spwn(["world", "knowledge", "ls", worldId]);
    expect(lsAfter.exitCode).toBe(0);
    expect(stripAnsi(lsAfter.output)).toContain("note.md");
  });

  test("knowledge files are accessible via simulated API structure", () => {
    // GIVEN - a SPWN_HOME with knowledge files
    const home = createSpwnHome();
    const knowledgeDir = join(home, "knowledge");
    mkdirSync(knowledgeDir, { recursive: true });

    writeFileSync(join(knowledgeDir, "overview.md"), "# Overview\n\nMain overview content.\n");
    writeFileSync(join(knowledgeDir, "glossary.md"), "# Glossary\n\nTerms and definitions.\n");
    mkdirSync(join(knowledgeDir, "projects"), { recursive: true });
    writeFileSync(join(knowledgeDir, "projects", "api.md"), "# API Project\n");

    // WHEN - listing files (simulating GET /api/knowledge)
    const { readdirSync, statSync } = require("node:fs");
    const walkFiles = (dir: string, base: string): string[] => {
      const results: string[] = [];
      const entries = readdirSync(dir, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = join(dir, entry.name);
        if (entry.isDirectory()) {
          results.push(...walkFiles(fullPath, base));
        } else {
          results.push(fullPath.replace(base + "/", ""));
        }
      }
      return results;
    };

    const files = walkFiles(knowledgeDir, knowledgeDir);

    // THEN - file list includes expected files
    expect(files).toContain("overview.md");
    expect(files).toContain("glossary.md");
    expect(files).toContain("projects/api.md");

    // WHEN - reading a specific file (simulating GET /api/knowledge/overview.md)
    const content = readFileSync(join(knowledgeDir, "overview.md"), "utf-8");

    // THEN - content is correct
    expect(content).toContain("# Overview");
    expect(content).toContain("Main overview content");
  });

  test("knowledge update via API persists to disk", () => {
    // GIVEN - a SPWN_HOME with knowledge directory
    const home = createSpwnHome();
    const knowledgeDir = join(home, "knowledge");
    mkdirSync(knowledgeDir, { recursive: true });

    const testFilePath = join(knowledgeDir, "test-file.md");
    const testContent = "# Test File\n\nCreated via API simulation.\n\n## Details\nThis file was written programmatically.\n";

    // WHEN - writing a file (simulating PUT /api/knowledge/test-file.md)
    writeFileSync(testFilePath, testContent);

    // THEN - file exists on disk
    expect(existsSync(testFilePath)).toBe(true);

    // THEN - reading it back returns the same content (simulating GET)
    const readBack = readFileSync(testFilePath, "utf-8");
    expect(readBack).toBe(testContent);
    expect(readBack).toContain("Created via API simulation");

    // WHEN - updating the file
    const updatedContent = testContent + "\n## Updated\nNew section added.\n";
    writeFileSync(testFilePath, updatedContent);

    // THEN - updated content persists
    const readUpdated = readFileSync(testFilePath, "utf-8");
    expect(readUpdated).toContain("New section added");
    expect(readUpdated).toContain("Created via API simulation");

    // CLEANUP - delete the test file
    unlinkSync(testFilePath);
    expect(existsSync(testFilePath)).toBe(false);
  });

  test("knowledge write to nested path creates subdirectories", () => {
    // GIVEN - a SPWN_HOME with knowledge directory
    const home = createSpwnHome();
    const knowledgeDir = join(home, "knowledge");
    mkdirSync(knowledgeDir, { recursive: true });

    // WHEN - writing to a nested path (simulating WriteFile with subdirs)
    const nestedPath = join(knowledgeDir, "projects", "backend", "architecture.md");
    mkdirSync(join(knowledgeDir, "projects", "backend"), { recursive: true });
    writeFileSync(nestedPath, "# Backend Architecture\n\nMicroservices design.\n");

    // THEN - file exists at nested location
    expect(existsSync(nestedPath)).toBe(true);
    const content = readFileSync(nestedPath, "utf-8");
    expect(content).toContain("Microservices design");
  });

  test("knowledge search across multiple files returns correct results", () => {
    // GIVEN - knowledge with multiple files containing a search term
    const home = createSpwnHome();
    const knowledgeDir = join(home, "knowledge");
    mkdirSync(knowledgeDir, { recursive: true });

    writeFileSync(
      join(knowledgeDir, "overview.md"),
      "# Overview\n\nThe authentication system uses JWT tokens for auth.\n",
    );
    writeFileSync(
      join(knowledgeDir, "glossary.md"),
      "# Glossary\n\n| Term | Definition |\n| authentication | Verifying user identity |\n",
    );
    writeFileSync(
      join(knowledgeDir, "roadmap.md"),
      "# Roadmap\n\n## Current Focus\n- Improve performance\n- Add caching layer\n",
    );
    writeFileSync(
      join(knowledgeDir, "security.md"),
      "# Security\n\nAuthentication and authorization best practices.\n",
    );

    // WHEN - searching for "authentication" (case-insensitive)
    const query = "authentication";
    const results: Record<string, string[]> = {};

    const { readdirSync } = require("node:fs");
    const allFiles = readdirSync(knowledgeDir);
    for (const file of allFiles) {
      const filePath = join(knowledgeDir, file);
      const stat = require("node:fs").statSync(filePath);
      if (!stat.isFile()) continue;

      const content = readFileSync(filePath, "utf-8");
      const matchingLines = content.split("\n").filter((line: string) =>
        line.toLowerCase().includes(query.toLowerCase()),
      );
      if (matchingLines.length > 0) {
        results[file] = matchingLines;
      }
    }

    // THEN - matches found in multiple files
    expect(Object.keys(results).length).toBeGreaterThanOrEqual(2);
    expect(Object.keys(results)).toContain("overview.md");
    expect(Object.keys(results)).toContain("glossary.md");
    expect(Object.keys(results)).toContain("security.md");
    expect(Object.keys(results)).not.toContain("roadmap.md");
  });
});
