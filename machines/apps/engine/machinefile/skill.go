package machinefile

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: the most bytes the machine's `spwn` skill holds, so an agent that
// loads it keeps its context for the work.

const SkillMax = 8 << 10

// value: the most bytes the drive section holds, five eighths of
// SkillMax: the rest is the head, where the machine is and its apps.

const DriveMax = 5 << 10

// value: the most bytes each text the skill takes holds: a name (a
// package, a command, a machine, a category), a line (an agent line, a
// description, a Machinefile value); and the most Machinefile values it
// says, the rest counted.
const (
	nameMax   = 64
	lineMax   = catalog.AgentMax
	valuesMax = 6
)

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
// package wrote for agents with the rest by name, or every app by name
// as far as the bound lets, the rest counted.
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
	// time, and the skill says how to read one app's line. It never holds
	// more than SkillMax bytes, whatever it is handed: each text it takes is
	// passes through oneLine, the drive section is clipped at DriveMax, and the last step
	// names the apps that fit and counts the rest.
	sorted := slices.Clone(present)
	slices.SortFunc(sorted, func(a, b catalog.App) int {
		return cmp.Or(cmp.Compare(rank(a.Category), rank(b.Category)), cmp.Compare(a.Name, b.Name))
	})
	drive = clip(strings.TrimSpace(strings.ToValidUTF8(drive, "")), DriveMax)
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

// invariant: every text the skill takes from a package, a Machinefile or
// the machine passes through oneLine before it is written: each control character and
// each blank (a newline, a tab, a line separator) becomes one space, the
// spaces fold to one, a format character and a backtick are dropped, it
// is trimmed, then clipped to limit bytes on a rune boundary. Each such text
// is written after a list marker and a label, never at a line's start, so
// none can open a heading, a front matter, a rule or a paragraph of its own.
func oneLine(text string, limit int) string {
	text = strings.Map(func(r rune) rune {
		switch {
		case r == '`' || unicode.Is(unicode.Cf, r):
			return -1
		case unicode.IsControl(r) || unicode.IsSpace(r):
			return ' '
		}
		return r
	}, strings.ToValidUTF8(text, ""))
	return clip(strings.Join(strings.Fields(text), " "), limit)
}

func clip(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return strings.TrimSpace(text[:cut])
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
	tail := "\n## How to drive spwn\n\n" + drive + "\n"
	switch {
	case len(present) == 0:
		b.WriteString("\nNone reads present yet.\n")
	case l == namesOnly:
		fitNames(&b, present, SkillMax-len(tail))
	default:
		lines(&b, present, l)
	}
	b.WriteString(tail)
	return b.String()
}

func title(c catalog.Category) string { return oneLine(cmp.Or(string(c), "Other"), nameMax) }

func lines(b *strings.Builder, present []catalog.App, l listing) {
	for i := 0; i < len(present); {
		c := present[i].Category
		j := i
		for j < len(present) && present[j].Category == c {
			j++
		}
		b.WriteString("\n### " + title(c) + "\n\n")
		var named []string
		for _, a := range present[i:j] {
			if l == agentLines && a.Agent == "" {
				named = append(named, "`"+oneLine(a.Name, nameMax)+"`")
				continue
			}
			b.WriteString("- " + appLine(a) + "\n")
		}
		if len(named) > 0 {
			lead := "- "
			if len(named) < j-i {
				lead = "- also "
			}
			b.WriteString(lead + strings.Join(named, ", ") + "\n")
		}
		i = j
	}
}

// invariant: the last listing names the apps in order while the skill,
// its drive section and the line that counts the rest fit within room
// bytes; every app it cannot name is counted on that line.
func fitNames(b *strings.Builder, present []catalog.App, room int) {
	room -= len(more(len(present))) + 1
	listed := 0
	for i, a := range present {
		add := ", "
		if i == 0 || a.Category != present[i-1].Category {
			add = "\n\n### " + title(a.Category) + "\n\n- "
			if i == 0 {
				add = add[1:]
			}
		}
		add += "`" + oneLine(a.Name, nameMax) + "`"
		if b.Len()+len(add)+1 > room {
			break
		}
		b.WriteString(add)
		listed++
	}
	if listed > 0 {
		b.WriteString("\n")
	}
	if rest := len(present) - listed; rest > 0 {
		if listed == 0 {
			b.WriteString("\n")
		}
		b.WriteString(more(rest))
	}
}

func more(n int) string {
	return "- and " + strconv.Itoa(n) + " more: `station apps` lists them\n"
}

func appLine(a catalog.App) string {
	name := "`" + oneLine(a.Name, nameMax) + "`"
	var cmds []string
	for _, c := range a.Provides {
		if c != a.Name {
			cmds = append(cmds, oneLine(c, nameMax))
		}
	}
	if len(cmds) > 0 {
		name += " (" + clip(strings.Join(cmds, ", "), lineMax) + ")"
	}
	return name + ": " + oneLine(cmp.Or(a.Agent, a.Description), lineMax)
}

func whereLines(w Where) []string {
	out := []string{"Machine: `" + oneLine(w.Name, nameMax) + "`, " + kindWords(w.Kind)}
	if sys := systemWords(w.System); sys != "" {
		out = append(out, "System: "+sys)
	}
	var values []string
	for _, v := range w.Values {
		if v.Key != "vm.mounts" && !strings.HasPrefix(v.Key, "ports.") {
			continue
		}
		values = append(values, "`"+oneLine(v.Key, nameMax)+"`: "+oneLine(v.Text, lineMax))
	}
	if len(values) > valuesMax {
		rest := len(values) - valuesMax
		values = append(values[:valuesMax], "and "+strconv.Itoa(rest)+" more values: `station machinefile explain` says each")
	}
	return append(out, values...)
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
	if v := oneLine(s.Version, nameMax); v != "" {
		words += " " + v
	}
	if a := oneLine(string(s.Arch), nameMax); a != "" {
		words += ", " + a
	}
	return words
}
