package overview

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

type Summary struct {
	// value: SPEC §5.1, a machine's Overview: its first line, Needs you, then
	// one line per section in the one order. It owns no fact: each line and
	// each finding is a section read's own.
	Head   Head
	Engine Engine
	Needs  []Need
	Lines  []Line
	// value: the VMs of a Mac, servers then workspaces, and the sentence of
	// its machines' foot ("3 running, 1 stopped"), absent with no VM.
	VMs      []VM
	Machines string `json:",omitempty"`
	// value: a section has not answered yet.
	Pending bool
	// value: SPEC §7.6's Mac readiness, empty on a Mac that needs none.
	Readiness []Check
	// value: the head of the page, from the engine, the needs and the
	// machines; and when this reading was made.
	Verdict verdict.Verdict
	Read    time.Time `json:",omitzero"`
}

type Head struct {
	// value: the machine an Overview is of, as its first line names it.
	Name    string
	Kind    machine.Kind
	ThisMac bool
	// value: how long the machine has run, as Activity's meter reads it;
	// zero where unknown.
	Up time.Duration
	// value: a Mac itself in one verdict: online with the machines it
	// runs, or its engine stopped; Summary.Verdict is what needs you. Absent
	// on a VM, whose head machines.list says from its Mac.
	Verdict *verdict.Verdict `json:",omitempty"`
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
	Machine string
	Section section.ID
	Broken  bool
	Subject string
	Text    string
	Call    string
	Args    map[string]string
	// value: the machine Call runs at, and the name its typed question
	// takes; empty with no Call, and with no typed question.
	On       string
	Confirm  string
	Command  string
	Path     string
	Severity finding.Severity
	// value: whether it needs the owner now or can wait, and the word of
	// its gesture.
	Upkeep Upkeep
	Verb   Verb
	// value: the word its line wears: Exposed for a port, Down for a
	// server, else its Upkeep's.
	Mark Mark
	// value: the sentence under its title, from its finding; empty when
	// the finding tells none.
	Detail string
	// value: the noun its line names after its machine, the tab its gesture
	// opens: its section's word, Sign-ins for a sign-in.
	Noun string
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

const LineStopped = "Start it to read its pages."

func Stopped(name string, kind machine.Kind) Summary {
	// value: the Overview of a VM that is stopped, as its Mac answers it
	// from its record: nothing read inside, its head saying why.
	return Summary{Head: Head{Name: name, Kind: kind}, Lines: []Line{}, Needs: []Need{}, Verdict: verdict.Verdict{Tone: verdict.Plain, Sentence: name + " is stopped.", Line: LineStopped}}
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
	for i := range all {
		all[i].Urgency = urgencyOf(all[i])
	}
	for _, f := range finding.Rank(all) {
		verb, command, path := VerbOf(f), f.Fix.Command, f.Fix.Path
		if verb == VerbReview {
			// invariant: a need that is reviewed opens its tab and carries
			// no command; where to change it stays on its row there.
			command, path = "", ""
		}
		s.Needs = append(s.Needs, Need{Machine: f.Machine.Name, Section: f.Section, Broken: f.Severity == finding.Broken, Severity: f.Severity,
			Subject: f.Subject, Text: f.Text, Call: f.Fix.Call, Args: f.Fix.Args, On: onOf(f), Confirm: f.Fix.Confirm, Command: command, Path: path,
			Upkeep: UpkeepOf(f), Verb: verb, Noun: NounOf(f.Section), Mark: MarkOf(f, vms), Detail: f.Detail})
	}
	s.Verdict, s.Machines = VerdictOf(s), MachinesOf(vms)
	s.Head.Verdict = HeadOf(s)
	return s
}

func onOf(f finding.Finding) string {
	if f.Fix.Call == "" {
		return ""
	}
	return f.Target()
}

func MachinesOf(vms []VM) string {
	running := 0
	for _, vm := range vms {
		if vm.Running {
			running++
		}
	}
	var parts []string
	if running > 0 {
		parts = append(parts, strconv.Itoa(running)+" running")
	}
	if stopped := len(vms) - running; stopped > 0 {
		parts = append(parts, strconv.Itoa(stopped)+" stopped")
	}
	return strings.Join(parts, ", ")
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
