package overview

import (
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
)

type Summary struct {
	// value: SPEC §5.1, a machine's Overview: its first line, Needs you, then
	// one line per section in the one order. It owns no fact: each line and
	// each finding is a section read's own.
	Head   Head
	Engine Engine
	Needs  []Need
	Lines  []Line
	// value: the VMs of a Mac, servers then workspaces.
	VMs []VM
	// value: a section has not answered yet.
	Pending bool
	// value: SPEC §7.6's Mac readiness, empty on a Mac that needs none.
	Readiness []Check
}

type Head struct {
	// value: the machine an Overview is of, as its first line names it.
	Name    string
	Kind    machine.Kind
	ThisMac bool
	// value: how long the machine has run, as Activity's meter reads it;
	// zero where unknown.
	Up time.Duration
}

func (h Head) Mac() bool    { return h.Kind == machine.Computer }
func (h Head) Server() bool { return h.Kind == machine.Server }

type Engine string

const (
	EngineRunning Engine = "running"
	EngineStopped Engine = "stopped"
	// value: a machine whose engine is not its own (a VM's is its Mac's).
	EngineNone Engine = ""
)

// value: the subject of the Needs-you line an owner's refused file is.

const RefusedSubject = "catalog"

type Need struct {
	// value: one Needs-you line, a section's finding as the pages draw it.
	Machine  string
	Section  section.ID
	Broken   bool
	Subject  string
	Text     string
	Call     string
	Args     map[string]string
	Command  string
	Path     string
	Severity finding.Severity
}

type Line struct {
	Section section.ID
	Parts   []Part
	Pending bool
	// value: why the section was not read, in words ("needs the engine").
	Unread string
}

type Part struct {
	// value: one part of a line: a count by state ("● 39 ok") when Counted
	// and State are set; a fact with its state ("● until 18:00") when only
	// State is; a plain count ("Private 2") or fact ("lid closed") without.
	State   catalog.State
	N       int
	Word    string
	Counted bool
	// value: a size the part leads with ("388.0 GB free").
	Bytes int64
	// value: a count's name for scripts: the section's own word for the
	// state it counts (apps' "update-available").
	Key string
	// value: the whole a count is part of ("● 12 of 17"), its Word then the
	// counted state's name for scripts.
	Of int
}

type VM struct {
	Name    string
	Running bool
}

type Read struct {
	// value: one section's read, as the composition hands it on.
	Line     Line
	Findings []finding.Finding
}

func Summarize(m Head, engine Engine, own []finding.Finding, reads []Read, vms []VM) Summary {
	// invariant: the lines keep the one order of the machine's kind; Needs
	// you is every read's findings and the Overview's own, ranked (§5.1.1).
	s := Summary{Head: m, Engine: engine, VMs: vms, Lines: []Line{}, Needs: []Need{}}
	all := append([]finding.Finding{}, own...)
	for _, id := range section.For(m.Kind) {
		for _, r := range reads {
			if r.Line.Section != id {
				continue
			}
			s.Lines = append(s.Lines, r.Line)
			s.Pending = s.Pending || r.Line.Pending
			all = append(all, r.Findings...)
		}
	}
	for _, r := range reads {
		if r.Line.Section == "" {
			all = append(all, r.Findings...)
		}
	}
	for _, f := range finding.Rank(all) {
		s.Needs = append(s.Needs, Need{Machine: f.Machine.Name, Section: f.Section, Broken: f.Severity == finding.Broken, Severity: f.Severity,
			Subject: f.Subject, Text: f.Text, Call: f.Fix.Call, Args: f.Fix.Args, Command: f.Fix.Command, Path: f.Fix.Path})
	}
	return s
}

func Count(state catalog.State, n int, word, key string) Part {
	return Part{State: state, N: n, Word: word, Key: key, Counted: true}
}

func Pending(id section.ID) Read {
	return Read{Line: Line{Section: id, Pending: true}}
}

func Unread(id section.ID, why string) Read {
	return Read{Line: Line{Section: id, Unread: why}}
}
