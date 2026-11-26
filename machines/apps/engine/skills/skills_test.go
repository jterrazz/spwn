package skills_test

import (
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/skills"
)

func TestANameIsLowercaseAndNoVerb(t *testing.T) {
	for name, want := range map[string]bool{
		"focus": true, "paper-reading": true, "a": true, strings.Repeat("a", skills.MaxName): true,
		"": false, "Focus": false, "2fast": false, "-focus": false, "a/b": false, "..": false, "focus.md": false,
		strings.Repeat("a", skills.MaxName+1): false, "create": false, "delete": false, "help": false,
	} {
		if got := skills.ValidName(name); got != want {
			t.Errorf("ValidName(%q) = %v; want %v", name, got, want)
		}
	}
}

func TestADescriptionIsReadFromTheFrontmatterAlone(t *testing.T) {
	for body, want := range map[string]string{
		"---\nname: a\ndescription: Reads papers.\n---\n# a\n": "Reads papers.",
		"---\ndescription: \"Reads: papers\"\n---\n":           "Reads: papers",
		"---\nname: a\n---\ndescription: not frontmatter\n":    "",
		"# a\n\ndescription: not frontmatter\n":                "",
		"---\nname: a\ndescription: never closed":              "never closed",
		"": "",
	} {
		if got := skills.Describe(body); got != want {
			t.Errorf("Describe(%q) = %q; want %q", body, got, want)
		}
	}
}

func TestANewSkillIsItsTemplateAndReadsBackItsDescription(t *testing.T) {
	s := skills.Blank("focus", "~/.spwn/skills/focus.md")
	if !strings.HasPrefix(s.Body, "---\nname: focus\ndescription: ") {
		t.Errorf("the template opens with %q; want the frontmatter the project check reads", s.Body)
	}
	if s.Name != "focus" || s.Description != "One-line description of what this skill does." || !strings.Contains(s.Body, "# focus") || !strings.Contains(s.Body, "## When to use") || !strings.Contains(s.Body, "## Steps") {
		t.Errorf("Blank = %+v; want focus with its template", s)
	}
}
