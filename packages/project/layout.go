package project

import (
	"path/filepath"
	"strings"

	"spwn.sh/packages/platform"
	"spwn.sh/packages/project/internal/discovery"
	"spwn.sh/packages/project/internal/layout"
	intmanifest "spwn.sh/packages/project/internal/manifest"
)

// Layout is where, from a project's root, the folder of each kind of what
// the project declares sits: agents/, skills/, tools/, hooks/, commands/
// and profiles/, each renamed by spwn.yaml's folders: when it says so.
type Layout = layout.Layout

// Kind is one kind of what a project declares, a key of folders:.
type Kind = layout.Kind

const (
	KindAgents   = layout.Agents
	KindSkills   = layout.Skills
	KindTools    = layout.Tools
	KindHooks    = layout.Hooks
	KindCommands = layout.Commands
	KindProfiles = layout.Profiles
)

// Kinds is every kind, in the order a walk of a project visits their
// folders.
var Kinds = layout.Kinds

// DefaultLayout is the layout of a project whose spwn.yaml renames no
// folder.
func DefaultLayout() Layout { return layout.Default() }

// NewLayout is the layout a folders: map declares; an entry it refuses is
// a *LayoutError naming its key.
func NewLayout(folders map[string]string) (Layout, error) { return layout.New(folders) }

// LayoutError refuses a folders: entry of spwn.yaml, naming its key.
type LayoutError = layout.Error

// LayoutCode is the code of a LayoutError.
const LayoutCode = layout.Code

// LayoutAt reads the layout of the project whose root is root from its
// spwn.yaml, through a root opened at root, before anything else of it
// is read.
func LayoutAt(root string) (Layout, error) {
	t := platform.Within(root)
	defer t.Close()
	if err := oldLayout(t, root); err != nil {
		return Layout{}, err
	}
	m, err := intmanifest.Read(t, filepath.Join(root, discovery.ManifestFileName))
	if err != nil {
		return Layout{}, err
	}
	return layout.New(m.Folders)
}

// Within runs act with root as the process's project, at the folders its
// spwn.yaml lays out, as platform.WithProject holds it; an empty root runs
// act in the user's own home. A layout spwn.yaml refuses is act's error,
// and act never runs.
func Within[T any](root string, act func() (T, error)) (T, error) {
	if root == "" {
		return platform.WithProject("", platform.Folders{}, act)
	}
	l, err := LayoutAt(root)
	if err != nil {
		var zero T
		return zero, err
	}
	return platform.WithProject(root, l.Platform(), act)
}

// OldLayoutCode is the code of the refusal of a project whose folders
// still sit under spwn/.
const OldLayoutCode = "project.old-layout"

// OldLayoutError refuses a project whose root holds the old layout's
// spwn/ folder.
type OldLayoutError struct {
	// Root is the project's root.
	Root string
	// AtRoot is each kind whose folder sits both under spwn/ and at the
	// root, in the order of Kinds.
	AtRoot []string
}

const oldLayoutSentence = "spwn/ is the old project layout: its folders now sit at the project's root. Move each one up (mv spwn/agents agents, and the same for skills, tools, hooks, commands, profiles), point knowledge: at its new path, change any .gitignore line naming spwn/, then run again."

func (e *OldLayoutError) Error() string { return OldLayoutCode + ": " + e.Message() }

// Message is the refusal without its code: the sentence that says how to
// move the project, and which folders already sit at the root.
func (e *OldLayoutError) Message() string {
	msg := oldLayoutSentence
	if len(e.AtRoot) == 0 {
		return msg
	}
	folders := make([]string, len(e.AtRoot))
	for i, k := range e.AtRoot {
		folders[i] = k + "/"
	}
	verb := "sits"
	if len(folders) > 1 {
		verb = "sit"
	}
	return msg + " " + strings.Join(folders, ", ") + " already " + verb + " at the root: merge what spwn/ holds of each into it, then remove spwn/."
}

// Code is OldLayoutCode.
func (e *OldLayoutError) Code() string { return OldLayoutCode }

// value: the names the old layout kept under spwn/, any of which makes a
// spwn/ folder at the root the old layout.
var oldNames = []string{"agents", "skills", "tools", "hooks", "commands", "profiles", "hooks.yaml"}

// invariant: a project whose root holds a spwn/ folder of the old layout is
// refused before its manifest is read, so no part of it is read from a
// folder it no longer means; a link named spwn/ is not followed.
func oldLayout(t *platform.Tree, root string) error {
	old := filepath.Join(root, "spwn")
	info, err := t.Lstat(old)
	if err != nil || !info.IsDir() {
		return nil
	}
	entries, err := t.ReadDir(old)
	if err != nil {
		return nil
	}
	held := false
	for _, e := range entries {
		for _, n := range oldNames {
			if e.Name() == n {
				held = true
			}
		}
	}
	if !held {
		return nil
	}
	refusal := &OldLayoutError{Root: root}
	for _, k := range layout.Kinds {
		_, errUnder := t.Lstat(filepath.Join(old, string(k)))
		_, errAt := t.Lstat(filepath.Join(root, string(k)))
		if errUnder == nil && errAt == nil {
			refusal.AtRoot = append(refusal.AtRoot, string(k))
		}
	}
	return refusal
}
