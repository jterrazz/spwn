// Package layout names the folder, from a project's root, that holds each
// kind of what a project declares: its agents, skills, tools, hooks,
// commands and profiles.
package layout

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"spwn.sh/packages/platform"
)

// Kind is one kind of what a project declares, and the key of spwn.yaml's
// folders: map that renames its folder.
type Kind string

const (
	Agents   Kind = "agents"
	Skills   Kind = "skills"
	Tools    Kind = "tools"
	Hooks    Kind = "hooks"
	Commands Kind = "commands"
	Profiles Kind = "profiles"
)

// Kinds is every kind, in the order a walk of a project visits their
// folders.
var Kinds = []Kind{Agents, Skills, Tools, Hooks, Commands, Profiles}

// Code is the code of a folders: entry of spwn.yaml that names no folder
// a project may hold.
const Code = "spwn-yaml.folders"

// Layout is where each kind's folder sits, as a slash path from the
// project's root: the kind's own name unless spwn.yaml's folders: renames
// it.
type Layout struct {
	folders map[Kind]string
}

// Default is the layout of a project whose spwn.yaml renames no folder.
func Default() Layout {
	folders := make(map[Kind]string, len(Kinds))
	for _, k := range Kinds {
		folders[k] = string(k)
	}
	return Layout{folders: folders}
}

// Error refuses a folders: entry of spwn.yaml, naming its key.
type Error struct {
	Key string
	msg string
}

func (e *Error) Error() string { return Code + ": " + e.msg }

// Message is the refusal without its code.
func (e *Error) Message() string { return e.msg }

// Code is the code every Error carries.
func (e *Error) Code() string { return Code }

// value: one segment of a folder's path: a lowercase letter or a digit,
// then letters, digits, - and _, so never "." or "..", never a leading dot,
// and never a /, a \, a ~, a : or a control character.
var segment = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// New is the layout spwn.yaml's folders: map declares over Default: each
// key a kind, each value the slash path of its folder from the root. An
// entry that is not a kind, or a value that is not a path of slug segments
// inside the project, or that is another kind's folder, under it or above
// it, is refused by an Error naming its key.
func New(folders map[string]string) (Layout, error) {
	l := Default()
	keys := make([]string, 0, len(folders))
	for k := range folders {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !isKind(key) {
			return Layout{}, &Error{Key: key, msg: fmt.Sprintf("folders.%s in spwn.yaml is not a kind: folders: renames one of %s", printable(key), kindList())}
		}
		value := folders[key]
		if why := refused(value); why != "" {
			return Layout{}, &Error{Key: key, msg: fmt.Sprintf("folders.%s in spwn.yaml is %q, %s: a folder is a path of the project from its root, its names lowercase letters, digits, - and _", key, value, why)}
		}
		l.folders[Kind(key)] = value
	}
	for i, a := range Kinds {
		for _, b := range Kinds[i+1:] {
			fa, fb := l.folders[a], l.folders[b]
			if fa == fb || strings.HasPrefix(fb, fa+"/") || strings.HasPrefix(fa, fb+"/") {
				key, met := b, a
				if _, declared := folders[string(a)]; declared {
					key, met = a, b
				}
				return Layout{}, &Error{Key: string(key), msg: fmt.Sprintf("folders.%s in spwn.yaml is %q, which meets %s/, the folder of %s: each kind has a folder of its own, none under another", key, l.folders[key], l.folders[met], met)}
			}
		}
	}
	return l, nil
}

func isKind(key string) bool {
	for _, k := range Kinds {
		if string(k) == key {
			return true
		}
	}
	return false
}

func kindList() string {
	names := make([]string, len(Kinds))
	for i, k := range Kinds {
		names[i] = string(k)
	}
	return strings.Join(names, ", ")
}

// value: key as an error may print it: a control character or a byte
// that is not UTF-8 quoted away.
func printable(key string) string {
	if segment.MatchString(key) {
		return key
	}
	return fmt.Sprintf("%q", key)
}

// value: why v is no folder of a project, or "" when it is one.
func refused(v string) string {
	switch {
	case v == "":
		return "empty"
	case !utf8.ValidString(v) || strings.ContainsFunc(v, func(r rune) bool { return r < 0x20 || (r >= 0x7f && r <= 0x9f) }):
		return "which holds a control character"
	case strings.HasPrefix(v, "/") || filepath.IsAbs(v):
		return "an absolute path"
	case strings.HasPrefix(v, "~"):
		return "which starts with ~, never expanded"
	case strings.Contains(v, ":"):
		return "which holds a colon"
	}
	for _, s := range strings.Split(v, "/") {
		switch {
		case s == "." || s == "..":
			return "which leaves its own path by " + s
		case strings.HasPrefix(s, "."):
			return "a hidden name, " + s
		case !segment.MatchString(s):
			return fmt.Sprintf("whose name %q is not a slug", s)
		}
	}
	if first, _, _ := strings.Cut(v, "/"); first == "spwn" {
		return "the old layout's folder, spwn/"
	}
	return ""
}

// Folder is the slash path, from the root, of k's folder; the zero Layout
// is Default.
func (l Layout) Folder(k Kind) string {
	if l.folders == nil {
		return string(k)
	}
	return l.folders[k]
}

// Dir is k's folder under root.
func (l Layout) Dir(root string, k Kind) string {
	return filepath.Join(root, filepath.FromSlash(l.Folder(k)))
}

// Folders is every kind's folder, as slash paths from the root, in the
// order of Kinds.
func (l Layout) Folders() []string {
	out := make([]string, len(Kinds))
	for i, k := range Kinds {
		out[i] = l.Folder(k)
	}
	return out
}

// Platform is l as the packages below this one take it.
func (l Layout) Platform() platform.Folders {
	return platform.Folders{
		Agents:   l.Folder(Agents),
		Skills:   l.Folder(Skills),
		Tools:    l.Folder(Tools),
		Hooks:    l.Folder(Hooks),
		Commands: l.Folder(Commands),
		Profiles: l.Folder(Profiles),
	}
}
