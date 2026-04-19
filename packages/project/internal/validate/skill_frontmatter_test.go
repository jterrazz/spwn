package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRuleSkillFrontmatter_detectsMissingBlock: a skill without any
// frontmatter is flagged with a fix-it hint. The spwn/skills/
// directory is the canonical bare-skill location.
func TestRuleSkillFrontmatter_detectsMissingBlock(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "spwn/skills", "naked.md", "# naked skill\n\ncontent\n")

	issues := ruleSkillFrontmatter(Input{Root: root})
	if len(issues) != 1 {
		t.Fatalf("want 1 issue for missing frontmatter, got %d: %+v", len(issues), issues)
	}
	if issues[0].Level != LevelError {
		t.Errorf("level: want Error, got %v", issues[0].Level)
	}
	if !strings.Contains(issues[0].Message, "missing YAML frontmatter") {
		t.Errorf("message: %q", issues[0].Message)
	}
	if !strings.Contains(issues[0].Hint, "name:") || !strings.Contains(issues[0].Hint, "description:") {
		t.Errorf("hint should show the expected header shape, got:\n%s", issues[0].Hint)
	}
}

// TestRuleSkillFrontmatter_passes: a well-formed skill produces no
// issues, whatever extra fields are present.
func TestRuleSkillFrontmatter_passes(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "spwn/skills", "good.md", `---
name: good
description: Use when the good path is required.
version: "1.0"
---

# body
`)
	issues := ruleSkillFrontmatter(Input{Root: root})
	if len(issues) != 0 {
		t.Errorf("want 0 issues, got %d: %+v", len(issues), issues)
	}
}

// TestRuleSkillFrontmatter_missingName: frontmatter present but
// name blank → error mentioning `name`.
func TestRuleSkillFrontmatter_missingName(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "spwn/skills", "nameless.md", `---
description: has everything but the name
---
body
`)
	issues := ruleSkillFrontmatter(Input{Root: root})
	if len(issues) != 1 {
		t.Fatalf("want 1 issue, got %d", len(issues))
	}
	if !strings.Contains(issues[0].Message, "name") {
		t.Errorf("message should mention name, got %q", issues[0].Message)
	}
}

// TestRuleSkillFrontmatter_missingDescription: name is declared but
// description is empty.
func TestRuleSkillFrontmatter_missingDescription(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "spwn/skills", "mute.md", `---
name: mute
---
body
`)
	issues := ruleSkillFrontmatter(Input{Root: root})
	if len(issues) != 1 {
		t.Fatalf("want 1 issue, got %d: %+v", len(issues), issues)
	}
	if !strings.Contains(issues[0].Message, "description") {
		t.Errorf("message should mention description, got %q", issues[0].Message)
	}
}

// TestRuleSkillFrontmatter_walksAuthoredSkillRoots: the rule covers
// every authoring location — project-wide (spwn/skills/) and per-tool
// (spwn/tools/<name>/skills/) — so a convention violation in either
// produces an issue.
func TestRuleSkillFrontmatter_walksAuthoredSkillRoots(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "spwn/skills", "project.md", "no header\n")
	writeSkill(t, root, "spwn/tools/my-tool/skills", "tool.md", "no header\n")

	issues := ruleSkillFrontmatter(Input{Root: root})
	if len(issues) != 2 {
		t.Fatalf("want 2 issues (one per authoring root), got %d: %+v", len(issues), issues)
	}
	// Issues are sorted by path (deterministic); verify each root is
	// represented so we don't silently drop one location.
	var paths []string
	for _, i := range issues {
		paths = append(paths, i.Path)
	}
	joined := strings.Join(paths, "|")
	for _, want := range []string{"spwn/skills/project.md", "spwn/tools/my-tool/skills/tool.md"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing coverage of %s in paths: %v", want, paths)
		}
	}
}

// TestRuleSkillFrontmatter_nestedSkills: a nested skill file (e.g.
// spwn/skills/reviewing/code.md) is still a skill. Walk is recursive.
func TestRuleSkillFrontmatter_nestedSkills(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "spwn/skills/reviewing", "code.md", "# no header\n")

	issues := ruleSkillFrontmatter(Input{Root: root})
	if len(issues) != 1 {
		t.Fatalf("nested skill should be checked, got %d issues", len(issues))
	}
	if !strings.Contains(issues[0].Path, "reviewing/code.md") {
		t.Errorf("nested path missing from report: %q", issues[0].Path)
	}
}

// TestRuleSkillFrontmatter_malformedYAML: invalid YAML inside a
// present block is a hard error (not a "missing frontmatter" one).
// Distinguishing them lets the user fix syntax separately from
// missing fields.
func TestRuleSkillFrontmatter_malformedYAML(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "spwn/skills", "bad.md", `---
name: [not-a-scalar
---
`)
	issues := ruleSkillFrontmatter(Input{Root: root})
	if len(issues) != 1 {
		t.Fatalf("want 1 issue for malformed YAML, got %d: %+v", len(issues), issues)
	}
	if !strings.Contains(issues[0].Message, "malformed") {
		t.Errorf("message should mark the block as malformed, got %q", issues[0].Message)
	}
}

func writeSkill(t *testing.T, root, dir, name, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(dir))
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(full, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
