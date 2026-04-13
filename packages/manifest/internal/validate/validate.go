// Package validate is the rule engine for spwn projects.
//
// Rules are pure functions: they take an Input and return zero or more
// Issues. Run calls every rule and returns the collected issues so a
// single `spwn check` invocation produces the full picture of what's
// wrong, not just the first failure.
package validate

import (
	"os"
	"path/filepath"
	"regexp"

	intmanifest "spwn.sh/packages/manifest/internal/manifest"
)

// Level ranks the severity of an Issue.
type Level int

const (
	// LevelError means the project cannot be used as-is. spwn up
	// would fail, or worse, silently do the wrong thing.
	LevelError Level = iota

	// LevelWarning means the project works but is missing something
	// the user probably wants.
	LevelWarning

	// LevelInfo is advisory — best-practice suggestions.
	LevelInfo
)

// String returns "error" / "warning" / "info" for display purposes.
func (l Level) String() string {
	switch l {
	case LevelError:
		return "error"
	case LevelWarning:
		return "warning"
	case LevelInfo:
		return "info"
	default:
		return "unknown"
	}
}

// Issue is one finding from a rule.
type Issue struct {
	// Level is the severity.
	Level Level

	// Path is a file path or a manifest field name — whatever best
	// identifies the location of the problem.
	Path string

	// Message is the human-readable description.
	Message string

	// Hint is an optional suggested fix, usually a command the user
	// can run. Empty if no obvious fix exists.
	Hint string
}

// Input is the data every rule operates on. Built by the public
// Validate function in packages/manifest and passed in verbatim.
type Input struct {
	Root        string
	Manifest    *intmanifest.Manifest
	AgentPaths  []string
	AgentExists []bool
	WorldPath   string
	WorldExists bool
}

// Run executes every rule against the input and returns all issues.
// Rules never short-circuit — we want a complete picture from one run.
func Run(in Input) []Issue {
	var out []Issue
	rules := []func(Input) []Issue{
		ruleManifestVersion,
		ruleManifestName,
		ruleManifestWorld,
		ruleManifestAgents,
		ruleWorldExists,
		ruleAgentDirs,
		ruleWorkspaceExists,
	}
	for _, r := range rules {
		out = append(out, r(in)...)
	}
	return out
}

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// --- Rules ---

func ruleManifestVersion(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	if in.Manifest.Version != intmanifest.CurrentVersion {
		return []Issue{{
			Level:   LevelError,
			Path:    "spwn.yaml#version",
			Message: "unsupported manifest version",
			Hint:    "set version: 1",
		}}
	}
	return nil
}

func ruleManifestName(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	if in.Manifest.Name == "" {
		return []Issue{{
			Level:   LevelError,
			Path:    "spwn.yaml#name",
			Message: "name is required",
			Hint:    "set name: to a slug like my-project",
		}}
	}
	if !slugRe.MatchString(in.Manifest.Name) {
		return []Issue{{
			Level:   LevelError,
			Path:    "spwn.yaml#name",
			Message: "name must match ^[a-z0-9][a-z0-9-]*$",
			Hint:    "use lowercase letters, digits, and dashes only",
		}}
	}
	return nil
}

func ruleManifestWorld(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	if in.Manifest.World == "" {
		return []Issue{{
			Level:   LevelError,
			Path:    "spwn.yaml#world",
			Message: "world is required",
			Hint:    "set world: to a name that matches ./spwn/worlds/<name>.yaml",
		}}
	}
	return nil
}

func ruleManifestAgents(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	if len(in.Manifest.Agents) == 0 {
		return []Issue{{
			Level:   LevelError,
			Path:    "spwn.yaml#agents",
			Message: "at least one agent must be declared",
			Hint:    "add an entry under agents: — one per directory under ./spwn/agents/",
		}}
	}
	return nil
}

func ruleWorldExists(in Input) []Issue {
	if in.WorldPath == "" {
		return nil
	}
	if in.WorldExists {
		return nil
	}
	return []Issue{{
		Level:   LevelError,
		Path:    relPath(in.Root, in.WorldPath),
		Message: "world config not found",
		Hint:    "create it with `spwn world new " + in.Manifest.World + "`",
	}}
}

func ruleAgentDirs(in Input) []Issue {
	if in.Manifest == nil {
		return nil
	}
	var out []Issue
	for i, name := range in.Manifest.Agents {
		if i >= len(in.AgentPaths) {
			continue
		}
		path := in.AgentPaths[i]
		exists := i < len(in.AgentExists) && in.AgentExists[i]
		if !exists {
			out = append(out, Issue{
				Level:   LevelError,
				Path:    relPath(in.Root, path),
				Message: "agent directory not found",
				Hint:    "create it with `spwn agent new " + name + "`",
			})
			continue
		}
		out = append(out, checkAgentStructure(in.Root, name, path)...)
	}
	return out
}

func checkAgentStructure(root, name, dir string) []Issue {
	required := []struct {
		rel   string
		isDir bool
		level Level
	}{
		{"agent.yaml", false, LevelError},
		{"CLAUDE.md", false, LevelError},
		{filepath.Join("core", "profile.md"), false, LevelError},
		{"core", true, LevelWarning},
		{"skills", true, LevelWarning},
		{"knowledge", true, LevelWarning},
		{"playbooks", true, LevelWarning},
		{"journal", true, LevelWarning},
	}
	var out []Issue
	for _, r := range required {
		full := filepath.Join(dir, r.rel)
		info, err := os.Stat(full)
		if err != nil {
			out = append(out, Issue{
				Level:   r.level,
				Path:    relPath(root, full),
				Message: "missing " + r.rel,
				Hint:    "regenerate with `spwn agent new " + name + " --force`",
			})
			continue
		}
		if r.isDir && !info.IsDir() {
			out = append(out, Issue{
				Level:   r.level,
				Path:    relPath(root, full),
				Message: r.rel + " is not a directory",
			})
		}
		if !r.isDir && info.IsDir() {
			out = append(out, Issue{
				Level:   r.level,
				Path:    relPath(root, full),
				Message: r.rel + " should be a file, found directory",
			})
		}
	}
	return out
}

func ruleWorkspaceExists(in Input) []Issue {
	if in.Manifest == nil || in.Manifest.Workspace == "" {
		return nil
	}
	workspace := in.Manifest.Workspace
	if !filepath.IsAbs(workspace) {
		workspace = filepath.Join(in.Root, workspace)
	}
	info, err := os.Stat(workspace)
	if err != nil {
		return []Issue{{
			Level:   LevelError,
			Path:    "spwn.yaml#workspace",
			Message: "workspace path not found: " + in.Manifest.Workspace,
			Hint:    "fix the workspace: field or create the directory",
		}}
	}
	if !info.IsDir() {
		return []Issue{{
			Level:   LevelError,
			Path:    "spwn.yaml#workspace",
			Message: "workspace must be a directory",
		}}
	}
	return nil
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
