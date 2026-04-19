// Package inspect implements the `spwn inspect` command — a
// kubectl-describe / cargo-tree blend that prints, per agent:
// identity header, resolved dependency tree with (*)-dedup and
// composition badges, skills list, and hooks list.
//
// Rendering is a pure function (Model → io.Writer) so golden tests
// pin the exact bytes without hitting disk or the Docker API.
package inspect

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"spwn.sh/packages/dependency/tool"
)

// Model is the top-level inspect document — one block per agent.
type Model struct {
	Agents []AgentView
}

// Status is the live runtime status of an agent's world. Empty
// (StatusUnknown) when the architect is unreachable; the renderer
// then shows "○ stopped" as the degraded default.
type Status string

const (
	StatusUnknown Status = ""
	StatusRunning Status = "running"
	StatusIdle    Status = "idle"
	StatusStopped Status = "stopped"
)

// AgentView carries everything one agent block needs.
type AgentView struct {
	Name    string
	Role    string
	Runtime string // stripped of spwn: prefix for display
	World   string
	Status  Status

	// Deps is the forest of top-level direct deps. Each DepNode
	// recurses through transitive deps with (*)-dedup.
	Deps []DepNode

	// Direct / transitive counters for the section header. Direct is
	// the number of top-level entries; transitive is the additional
	// deps pulled in by the graph walk.
	DirectDepsCount     int
	TransitiveDepsCount int

	Skills []SkillRef
	Hooks  []HookRef
}

// DepNode is one node in the dep tree. DedupSeen=true means this
// name was already rendered above and only the "(*)" placeholder
// should print (children are suppressed, cargo-tree convention).
type DepNode struct {
	Name      string
	Version   string
	Kind      tool.Kind
	Skills    int // count of markdown files in the tool's skills/ dir
	DedupSeen bool
	Children  []DepNode
}

// SkillRef is one entry under the Skills section. Origin is the
// short source label ("spwn/skills", "spwn:qmd", "my-parser").
type SkillRef struct {
	Name   string
	Origin string
}

// HookRef is one entry under the Hooks section.
type HookRef struct {
	Name   string
	Origin string
}

// Render writes the whole model to w. Multiple agents are separated
// by a single blank line — terraform/helm convention: whitespace is
// the only separator.
func Render(w io.Writer, m Model) {
	for i, a := range m.Agents {
		if i > 0 {
			fmt.Fprintln(w)
		}
		renderAgent(w, a)
	}
}

func renderAgent(w io.Writer, a AgentView) {
	// Header — kubectl-describe: Key <pad> Value, one per line.
	fmt.Fprintf(w, "Name         %s\n", a.Name)
	fmt.Fprintf(w, "Role         %s\n", a.Role)
	if a.Runtime != "" {
		fmt.Fprintf(w, "Runtime      %s\n", a.Runtime)
	}
	if a.World != "" {
		fmt.Fprintf(w, "World        %s   (%s)\n", a.World, renderStatus(a.Status))
	}

	// Dependencies section.
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Dependencies (%s)\n",
		depCountLabel(a.DirectDepsCount, a.TransitiveDepsCount))
	for _, root := range a.Deps {
		renderDep(w, root, 0)
	}

	// Skills section — two-column: Name<pad>Origin.
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Skills (%d)\n", len(a.Skills))
	if len(a.Skills) == 0 {
		fmt.Fprintf(w, "  -\n")
	} else {
		maxName := 0
		for _, s := range a.Skills {
			if n := visibleLen(s.Name); n > maxName {
				maxName = n
			}
		}
		for _, s := range a.Skills {
			pad := strings.Repeat(" ", maxName-visibleLen(s.Name)+4)
			fmt.Fprintf(w, "  %s%s%s\n", s.Name, pad, s.Origin)
		}
	}

	// Hooks section.
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Hooks (%d)\n", len(a.Hooks))
	if len(a.Hooks) == 0 {
		fmt.Fprintf(w, "  -\n")
	} else {
		maxName := 0
		for _, h := range a.Hooks {
			if n := visibleLen(h.Name); n > maxName {
				maxName = n
			}
		}
		for _, h := range a.Hooks {
			pad := strings.Repeat(" ", maxName-visibleLen(h.Name)+4)
			fmt.Fprintf(w, "  %s%s%s\n", h.Name, pad, h.Origin)
		}
	}
}

// renderDep prints one dep subtree. badgeCol is the column at which
// composition badges (skills(N) · config · runtime) start so they
// line up across siblings at the same depth.
func renderDep(w io.Writer, d DepNode, depth int) {
	indent := strings.Repeat("  ", depth+1)
	label := d.Name
	if d.Version != "" {
		label += "@" + d.Version
	}
	if d.DedupSeen {
		label += "  (*)"
	}
	line := indent + label
	badges := composeBadges(d)
	if badges != "" {
		line += strings.Repeat(" ", max(0, badgeColumn-visibleLen(line))) + badges
	}
	fmt.Fprintln(w, line)
	if d.DedupSeen {
		return
	}
	for _, c := range d.Children {
		renderDep(w, c, depth+1)
	}
}

// badgeColumn is the fixed column where composition badges start.
// Wide enough to cover "  spwn:paperclip-factory@latest" + one
// space of breathing room at depth 1; deeper nodes lose alignment
// only when the name is extraordinarily long.
const badgeColumn = 35

func composeBadges(d DepNode) string {
	var parts []string
	if d.Kind != tool.KindTool && d.Kind != "" {
		parts = append(parts, string(d.Kind))
	}
	if d.Skills > 0 {
		parts = append(parts, fmt.Sprintf("skills(%d)", d.Skills))
	}
	return strings.Join(parts, " · ")
}

func renderStatus(s Status) string {
	switch s {
	case StatusRunning:
		return "● running"
	case StatusIdle:
		return "◐ idle"
	default:
		return "○ stopped"
	}
}

func depCountLabel(direct, transitive int) string {
	return fmt.Sprintf("%d direct, %d transitive", direct, transitive)
}

// visibleLen strips the (*) dedup suffix from a line for padding
// math so dedup markers don't push badges out of alignment.
func visibleLen(s string) int {
	// No ANSI in inspect output (pipe-safe), so a rune count suffices.
	return len([]rune(s))
}

// sortSkills keeps skills deterministically ordered: project-local
// first (origin starts with "spwn/"), then catalog-provided (starts
// with "@"), then local tools. Within each group, alphabetical by
// origin then by name.
func sortSkills(s []SkillRef) {
	sort.SliceStable(s, func(i, j int) bool {
		pi, pj := skillOriginPriority(s[i].Origin), skillOriginPriority(s[j].Origin)
		if pi != pj {
			return pi < pj
		}
		if s[i].Origin != s[j].Origin {
			return s[i].Origin < s[j].Origin
		}
		return s[i].Name < s[j].Name
	})
}

func skillOriginPriority(origin string) int {
	switch {
	case strings.HasPrefix(origin, "spwn/"):
		return 0
	case strings.HasPrefix(origin, "@"):
		return 1
	default:
		return 2
	}
}
