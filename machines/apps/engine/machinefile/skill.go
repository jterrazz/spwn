package machinefile

import (
	"cmp"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: the most bytes the machine's `spwn` skill holds, so an agent that
// loads it keeps its context for the work.

const SkillMax = 8 << 10

// value: what the `spwn` skill says of the machine it is written on.

type Where struct {
	Name   string
	Kind   catalog.MachineRole
	System catalog.System
	// value: the machine's mounts and port rights as its resolution words
	// them (`vm.mounts`, `ports.*`), in the resolution's order.
	Values []Value
}

// value: how much of each app the skill says: every line, the lines a
// package wrote for agents with the rest by name, or every app by name.
type listing int

const (
	everyLine listing = iota
	agentLines
	namesOnly
)

func Skill(where Where, present []catalog.App, drive string) []byte {
	// invariant: the skill is the same bytes for the same machine, apps
	// and drive section: the apps stand in the catalog's category order,
	// then by name. Past SkillMax the apps are said shorter, a step at a
	// time, and the skill says how to read one app's line.
	sorted := slices.Clone(present)
	slices.SortFunc(sorted, func(a, b catalog.App) int {
		return cmp.Or(cmp.Compare(rank(a.Category), rank(b.Category)), cmp.Compare(a.Name, b.Name))
	})
	var body string
	for _, l := range []listing{everyLine, agentLines, namesOnly} {
		if body = skillText(where, sorted, drive, l); len(body) <= SkillMax {
			break
		}
	}
	return []byte(body)
}

func rank(c catalog.Category) int {
	if i := slices.Index(catalog.Categories, c); i >= 0 {
		return i
	}
	return len(catalog.Categories)
}

const skillHead = `---
name: spwn
description: The machine you are on, the apps installed on it (its command-line programs included) and when to use each, and how to drive spwn (the station command). Load it when you need a program, wonder whether one is installed, or ask what machine this is.
---

# spwn: this machine

Machines (` + "`station`" + `) writes this file at the end of every apply and update of the machine, from the machine's record, the apps that read present and its own commands. It is rewritten whole: edit nothing here.
`

func skillText(where Where, present []catalog.App, drive string, l listing) string {
	var b strings.Builder
	b.WriteString(skillHead)
	b.WriteString("\n## Where you are\n\n")
	for _, line := range whereLines(where) {
		b.WriteString("- " + line + "\n")
	}
	b.WriteString("\n## The apps present\n\n")
	switch l {
	case everyLine:
		b.WriteString("Each app by its package name, the commands it brings when they differ, and when to use it. `station apps <name>` prints one app's detail.\n")
	case agentLines:
		b.WriteString("The apps with a line for agents say it; the rest are named only, to keep this file within 8 KiB. `station apps <name>` prints one app's detail.\n")
	case namesOnly:
		b.WriteString("The apps by name only, to keep this file within 8 KiB. `station apps <name>` prints one app's detail.\n")
	}
	if len(present) == 0 {
		b.WriteString("\nNone reads present yet.\n")
	}
	for i := 0; i < len(present); {
		c := present[i].Category
		j := i
		for j < len(present) && present[j].Category == c {
			j++
		}
		b.WriteString("\n### " + cmp.Or(string(c), "Other") + "\n\n")
		var named []string
		for _, a := range present[i:j] {
			if l == namesOnly || l == agentLines && a.Agent == "" {
				named = append(named, "`"+a.Name+"`")
				continue
			}
			b.WriteString("- " + appLine(a) + "\n")
		}
		if len(named) > 0 {
			if l == agentLines && len(named) < j-i {
				b.WriteString("- also ")
			}
			b.WriteString(strings.Join(named, ", ") + "\n")
		}
		i = j
	}
	b.WriteString("\n## How to drive spwn\n\n")
	b.WriteString(strings.TrimSpace(drive) + "\n")
	return b.String()
}

func appLine(a catalog.App) string {
	name := "`" + a.Name + "`"
	if cmds := slices.DeleteFunc(slices.Clone(a.Provides), func(c string) bool { return c == a.Name }); len(cmds) > 0 {
		name += " (" + strings.Join(cmds, ", ") + ")"
	}
	return name + ": " + cmp.Or(a.Agent, a.Description)
}

func whereLines(w Where) []string {
	out := []string{"Machine: `" + w.Name + "`, " + kindWords(w.Kind)}
	if sys := systemWords(w.System); sys != "" {
		out = append(out, "System: "+sys)
	}
	for _, v := range w.Values {
		if v.Key != "vm.mounts" && !strings.HasPrefix(v.Key, "ports.") {
			continue
		}
		out = append(out, "`"+v.Key+"`: "+v.Text)
	}
	return out
}

func kindWords(k catalog.MachineRole) string {
	switch k {
	case catalog.RoleWorkspace:
		return "a workspace (a VM set up by Machines for you)"
	case catalog.RoleServer:
		return "a server (a VM kept always on)"
	case catalog.RoleComputer:
	}
	return "a computer (a Mac)"
}

func systemWords(s catalog.System) string {
	if s.OS == "" {
		return ""
	}
	words := s.OS.Word()
	if s.Version != "" {
		words += " " + s.Version
	}
	if s.Arch != "" {
		words += ", " + string(s.Arch)
	}
	return words
}
