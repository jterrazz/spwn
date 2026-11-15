package accounts

import (
	"strconv"
	"strings"

	model "github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

const KeyReload = "R"

func columns() []page.Column {
	return []page.Column{
		{Key: "program", Kind: page.KindText, Keep: 100},
		{Key: "state", Kind: page.KindText, Keep: 90},
		{Key: "who", Kind: page.KindIdentity, Keep: 60},
		{Key: "how", Kind: page.KindText, Keep: 70},
		{Key: "scope", Kind: page.KindList, Keep: 50, Whole: true},
		{Key: "line", Kind: page.KindText, Keep: 40},
	}
}

func mark(r model.ProgramRow) page.Mark {
	switch r.State {
	case model.ProgramSignedIn:
		return page.OK
	case model.ProgramBroken:
		return page.Broken
	case model.ProgramReading, model.ProgramWaiting:
		return page.Awaiting
	case model.ProgramSignedOut:
	}
	return page.Absent
}

func stateWord(r model.ProgramRow) string { return strings.ReplaceAll(string(r.State), "-", " ") }

func how(r model.ProgramRow) string {
	switch {
	case r.State != model.ProgramSignedIn:
		return ""
	case r.How == model.HowGranted && r.EntryLabel != "":
		return "granted: " + r.EntryLabel
	}
	return string(r.How)
}

func scopes(r model.ProgramRow) []string {
	if r.Plan != "" {
		return append([]string{r.Plan}, r.Scopes...)
	}
	return r.Scopes
}

func row(name string, r model.ProgramRow) page.Row {
	out := page.Row{ID: name + "/" + string(r.Program), Label: string(r.Program), Short: r.Program.Label(), Mark: mark(r)}
	who := page.None()
	if r.Who != "" {
		who = page.Identity(r.Who)
	}
	out.Cells = []page.Value{page.Text(stateWord(r)), who, page.Text(how(r)), page.Texts(scopes(r)...), page.Text(r.Line)}
	if out.Mark == page.Broken {
		out.Reason = page.Failing
	}
	return out
}

func section(id, title string, m Machine) page.Section {
	s := page.Section{ID: id, Title: title, Columns: columns()}
	for _, r := range m.Read.Programs {
		one := row(m.Name, r)
		s.Items = append(s.Items, page.Item{Row: &one})
	}
	if len(s.Items) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Noun: "programs"}}}
	}
	return s
}

type Machine struct {
	// value: one machine of the page: its accounts.list, or the line that
	// says why it was not read.
	Name string
	Read model.ProgramsRead
	Note string
	// value: a machine not read, marked as every surface marks it: a
	// stopped VM, a Mac that does not answer; State is its word.
	Mark   page.Mark
	Reason page.Reason
	State  string
	// value: how its document names it (SPEC §11.1), and why it was not
	// read: the type and code of the error it answered, or that it is a server.
	Label, Kind, On string
	ThisMac         bool
	Fault           Fault
	Server          bool
}

type Fault struct{ Type, Code string }

func Page(m Machine, thisMac page.Context) page.Page {
	p := page.Page{ID: "accounts", Command: "accounts", Context: thisMac, Sections: []page.Section{firstLine(m, thisMac), section("accounts", "Accounts", m)}}
	p.Actions = []page.Action{{Key: KeyReload, Label: "reload", On: page.OnPage, Priority: 1}}
	return page.Seal(p)
}

// value: the first line: the machine, its programs, how many are signed in
// and what needs you.
func firstLine(m Machine, thisMac page.Context) page.Section {
	label := "This Mac"
	if thisMac.Machine != "" && thisMac.Kind != string(machine.Computer) {
		label = thisMac.Machine
	}
	in, needs, reading := 0, 0, false
	for _, r := range m.Read.Programs {
		switch r.State {
		case model.ProgramSignedIn:
			in++
		case model.ProgramBroken:
			needs++
		case model.ProgramReading, model.ProgramWaiting:
			reading = true
		case model.ProgramSignedOut:
		}
	}
	var words []string
	switch len(m.Read.Programs) {
	case 0:
		words = append(words, "no program")
	case 1:
		words = append(words, "1 program", strconv.Itoa(in)+" signed in")
	default:
		words = append(words, strconv.Itoa(len(m.Read.Programs))+" programs", strconv.Itoa(in)+" signed in")
	}
	switch {
	case reading:
		words = append(words, "reading")
	case needs == 1:
		words = append(words, "1 needs you")
	case needs > 1:
		words = append(words, strconv.Itoa(needs)+" need you")
	}
	r := page.Row{ID: "summary/accounts", Label: label, Mark: page.OK, Cells: []page.Value{page.Texts(words...)}}
	return page.Section{ID: "summary", Title: "Summary", Role: page.Summary, Fold: page.Open, Uncounted: true, Columns: []page.Column{{Key: "area", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}, Items: []page.Item{{Row: &r}}}
}

func List(m Machine) page.Page {
	// value: the Accounts section as the CLI prints it piped: one row per
	// program of the machine's accounts.list.
	s := section("records", "Accounts", m)
	s.Role = page.Banner
	return page.Seal(page.Page{ID: "accounts", Command: "accounts", Sections: []page.Section{s}})
}

func EveryMachine(blocks []Machine) page.Page {
	var sections []page.Section
	for _, b := range blocks {
		switch {
		case b.Mark != "":
			r := page.Row{ID: b.Name, Label: b.Name, Mark: b.Mark, Reason: b.Reason, Cells: []page.Value{page.Text(b.State)}}
			sections = append(sections, page.Section{ID: b.Name, Title: b.Name, Items: []page.Item{{Row: &r}}})
		case b.Note != "":
			sections = append(sections, page.Section{ID: b.Name, Title: b.Name, Items: []page.Item{{Note: &page.Note{Text: b.Note}}}})
		default:
			sections = append(sections, section(b.Name, b.Name, b))
		}
	}
	return page.Seal(page.Page{ID: "accounts", Command: "accounts", Sections: sections})
}

type ListJSON struct {
	Schema   string         `json:"schema"`
	Machine  MachineRef     `json:"machine"`
	Findings []finding.JSON `json:"findings"`
	Accounts []AccountRow   `json:"accounts"`
}

type EveryJSON struct {
	Schema   string         `json:"schema"`
	Findings []finding.JSON `json:"findings"`
	Machines []MachineJSON  `json:"machines"`
	Skipped  []SkippedJSON  `json:"skipped"`
}

type MachineJSON struct {
	Machine  MachineName  `json:"machine"`
	Accounts []AccountRow `json:"accounts"`
	Error    *ErrorJSON   `json:"error"`
}

type MachineName struct {
	Name    string  `json:"name"`
	Label   string  `json:"label"`
	Kind    string  `json:"kind"`
	On      *string `json:"on"`
	ThisMac bool    `json:"this_mac"`
}

type ErrorJSON struct {
	Type    string  `json:"type"`
	Code    *string `json:"code"`
	Message string  `json:"message"`
}

type SkippedJSON struct {
	Name string `json:"name"`
	Why  string `json:"why"`
}

type MachineRef struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type AccountRow struct {
	Program    string `json:"program"`
	Label      string `json:"label"`
	State      string `json:"state"`
	Who        string `json:"who,omitempty"`
	Scope      string `json:"scope,omitempty"`
	How        string `json:"how,omitempty"`
	EntryLabel string `json:"entry_label,omitempty"`
	Line       string `json:"line,omitempty"`
}

func rows(m Machine) []AccountRow {
	// invariant: names, states, who and scopes only: no field of it ever
	// carries a token, a cookie or a key body.
	out := make([]AccountRow, 0, len(m.Read.Programs))
	for _, r := range m.Read.Programs {
		out = append(out, AccountRow{
			Program: string(r.Program), Label: r.Program.Label(), State: string(r.State), Who: r.Who,
			Scope: strings.Join(scopes(r), ", "), How: string(r.How), EntryLabel: r.EntryLabel, Line: r.Line,
		})
	}
	return out
}

func ListContract(schema string, m Machine) ListJSON {
	return ListJSON{Schema: schema, Machine: MachineRef{Name: m.Name, Kind: m.Kind}, Findings: finding.Doc(model.Findings(m.Read, machine.Ref{})), Accounts: rows(m)}
}

func EveryMachineContract(schema string, blocks []Machine) EveryJSON {
	// invariant: one entry per Mac and workspace read, in the order of the
	// page; one that did not answer keeps its place with its error, a
	// server is skipped and said (SPEC §11.1).
	out := EveryJSON{Schema: schema, Machines: []MachineJSON{}, Skipped: []SkippedJSON{}}
	var fs []finding.Finding
	for _, b := range blocks {
		if b.Server {
			out.Skipped = append(out.Skipped, SkippedJSON{Name: b.Name, Why: "server"})
			continue
		}
		name := MachineName{Name: b.Name, Label: b.Label, Kind: b.Kind, ThisMac: b.ThisMac}
		if b.On != "" {
			on := b.On
			name.On = &on
		}
		m := MachineJSON{Machine: name, Accounts: []AccountRow{}}
		if b.Mark != "" || b.Note != "" {
			e := ErrorJSON{Type: b.Fault.Type, Message: b.Note}
			if b.Fault.Code != "" {
				code := b.Fault.Code
				e.Code = &code
			}
			if e.Message == "" {
				e.Message = b.State
			}
			m.Error = &e
			out.Machines = append(out.Machines, m)
			continue
		}
		m.Accounts = rows(b)
		out.Machines = append(out.Machines, m)
		fs = append(fs, model.Findings(b.Read, machine.Ref{Name: b.Name, On: b.On})...)
	}
	out.Findings = finding.AllMachinesDoc(fs)
	return out
}
