package project

import (
	"errors"
	"regexp"
)

type Project struct {
	Name string
	// value: the folder its spwn.yaml sits in, as a path on its machine.
	Root string
	// value: its spwn.yaml, as a path on its machine.
	Manifest string
	// value: the agents its spwn/agents/ holds, by name.
	Agents []string
}

type Level string

const (
	Error   Level = "error"
	Warning Level = "warning"
	Info    Level = "info"
)

func (Level) Enum() []string { return []string{string(Error), string(Warning), string(Info)} }

type Issue struct {
	Level Level
	// value: what the issue is about: a file of the project, or a key of
	// one after `#`.
	Path    string
	Message string
	Hint    string
	// value: the rule that found it: manifest, the declaration's own rules;
	// compile, a render of each agent that writes nothing.
	Source string
}

const (
	SourceManifest = "manifest"
	SourceCompile  = "compile"
	// value: a drift row: a reference the lockfile does not record.
	SourceLockfile = "lockfile"
)

type Summary struct {
	Errors, Warnings, Info int
}

type Plan struct {
	Root     string
	Manifest string
	// value: no issue is an error, so the project can be applied.
	Valid   bool
	Summary Summary
	// value: the errors first, then the warnings, then the advice.
	Issues []Issue
	// value: what apply would write and remove, by path; none when the
	// plan has an error, as nothing is rendered then.
	Changes []Change
}

type Drift struct {
	Root     string
	Manifest string
	// value: each way what was installed for the project differs from what
	// it declares; none when they agree.
	Rows []Issue
}

type State struct {
	// invariant: what the world keeps of one project: the project, what its
	// plan finds, and how it drifted.
	Project Project
	Issues  []Issue `json:",omitempty"`
	Drift   []Issue `json:",omitempty"`
	// value: what each agent's harness renders, by agent.
	Rendered map[string][]File `json:",omitempty"`
	// value: what the run folders and the declaration hold, by path from
	// the root, beyond what the declaration's own events say.
	Disk map[string]File `json:",omitempty"`
	// value: the paths from the root that hold neither a file, a folder
	// nor a link: a FIFO, a device.
	Irregular []string `json:",omitempty"`
	// value: the folders from the root without their write bit, in which
	// no name is made, replaced or removed.
	ReadOnly []string `json:",omitempty"`
	// invariant: what apply last wrote, as Digest gives it.
	Record Record `json:",omitempty"`
}

var (
	ErrNotFound = errors.New("no project in that folder or above it")
	ErrExists   = errors.New("that folder exists")
)

const (
	// value: the file a folder is a project by.
	ManifestName = "spwn.yaml"
	// value: the agent a new project is made with.
	Starter = "neo"
)

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func ValidName(name string) bool {
	// invariant: the manifest's rule for a project's name: lowercase letters,
	// digits and dashes, a letter or a digit first.
	return namePattern.MatchString(name)
}

func PlanOf(root, manifest string, found []Issue) Plan {
	// invariant: the issues by level, each level in the order found; the
	// project is valid when none is an error.
	p := Plan{Root: root, Manifest: manifest, Issues: []Issue{}}
	for _, level := range []Level{Error, Warning, Info} {
		for _, i := range found {
			if i.Level == level {
				p.Issues = append(p.Issues, i)
			}
		}
	}
	for _, i := range p.Issues {
		switch i.Level {
		case Error:
			p.Summary.Errors++
		case Warning:
			p.Summary.Warnings++
		case Info:
			p.Summary.Info++
		}
	}
	p.Valid = p.Summary.Errors == 0
	return p
}
