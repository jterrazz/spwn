package skills

import (
	"errors"
	"regexp"
	"slices"
	"strings"
)

type Skill struct {
	Name string
	// value: the file the skill is, as a path on its machine.
	Path string
	// value: its one line of purpose, as the frontmatter of its file says it;
	// empty when the file has none.
	Description string
	// value: the whole file; empty in a list, which reads no file twice.
	Body string
}

var (
	ErrNotFound = errors.New("no skill of that name")
	ErrExists   = errors.New("a skill of that name exists")
	ErrBadName  = errors.New("not a name a skill takes")
)

// value: what a skill's name may be, and what its file's name ends with.
const (
	MaxName = 63
	Suffix  = ".md"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func Verbs() []string {
	// value: the words under `skills` that are verbs, which no skill may be
	// named, since `skills <name>` is that skill's detail.
	return []string{"create", "delete", "help"}
}

func ValidName(name string) bool {
	// invariant: a name is lowercase letters, digits and dashes, starts with
	// a letter, is at most MaxName long, and is no verb of the noun.
	return len(name) <= MaxName && namePattern.MatchString(name) && !slices.Contains(Verbs(), name)
}

func Describe(body string) string {
	// invariant: the description is the `description:` line of the frontmatter
	// the file opens with, quotes dropped; a file with no frontmatter has none.
	lines := strings.Split(body, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return ""
		}
		if value, ok := strings.CutPrefix(line, "description:"); ok {
			return strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return ""
}

func Template(name string) string {
	// value: the file a new skill is made as: its frontmatter, then the
	// sections a skill answers.
	return "---\nname: " + name + "\ndescription: One-line description of what this skill does.\n---\n\n# " + name + `

> One-line description of what this skill does.

## When to use

Describe the trigger. When should the agent invoke this skill?

## Prerequisites

List what it needs: programs, environment, or prior knowledge.

## Steps

1. First step
2. Second step
3. Third step

## Rollback

How to undo if things go wrong.
`
}

func Blank(name, path string) Skill {
	body := Template(name)
	return Skill{Name: name, Path: path, Description: Describe(body), Body: body}
}
