package activity

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/page"
)

const Schema = "station.activity.read/v1"

const (
	computer  = "computer"
	workspace = "workspace"
	server    = "server"
)

type View struct {
	// value: the machine read: its name, the label its page says, its kind,
	// the Mac it is on (empty for a Mac) by name and as the owner reads it.
	Name  string
	Label string
	// value: the name a terminal shows of a Mac its records name whole.
	Short string
	// value: "computer", "workspace" or "server".
	Kind    string
	Mac     string
	On      string
	ThisMac bool
	Report  activity.Report
	// value: a VM that does not run: it has no activity.
	Stopped bool
	// value: a Mac that does not answer, in its words, and the type and code
	// of the error it answered.
	Silent                 string
	SilentType, SilentCode string
	// value: the window the TUI's graphs draw; zero on the CLI's page,
	// which draws none (SPEC §6.4).
	Graph time.Duration
}

func silent() page.Value { return page.Texts("?", "no answer in 2 s") }

func columns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindText, Keep: 100}}
}

func listing() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindListing, Keep: 100}, {Key: "value", Kind: page.KindText, Keep: 100}}
}

func row(id, label string, mark page.Mark, words ...string) page.Item {
	r := page.Row{ID: id, Label: label, Mark: mark, Cells: []page.Value{page.Texts(words...)}}
	return page.Item{Row: &r}
}

func cell(id, label string, v page.Value) page.Item {
	r := page.Row{ID: id, Label: label, Cells: []page.Value{v}}
	return page.Item{Row: &r}
}

func head(v View) []string {
	r := v.Report
	var words []string
	if v.Kind != computer {
		words = append(words, kindWord(v.Kind))
	}
	switch run := r.Last.Run; {
	case r.Silent:
	case run != nil && !run.Started.IsZero():
		words = append(words, "up "+activity.Uptime(r.Last.Sample.At.Sub(run.Started))+" (since its run started)")
	case run == nil && !r.Last.Sample.Boot.IsZero():
		words = append(words, "up "+activity.Uptime(r.Last.Sample.At.Sub(r.Last.Sample.Boot)))
	}
	switch {
	case r.Silent:
		words = append(words, "the meter does not answer")
	case r.Source == activity.FromMacProcess:
		words = append(words, "from "+v.On+" (its guest does not answer)")
	case r.Source == activity.FromDirect && r.Window > 0:
		words = append(words, "rates over "+activity.Window(r.Window))
	case r.Window > 0:
		words = append(words, "rates over the last "+activity.Window(activity.RateWindow))
	default:
		words = append(words, "no rate yet")
	}
	return words
}

func kindWord(k string) string {
	switch k {
	case server:
		return "Server"
	case workspace:
		return "Workspace"
	}
	return "Mac"
}

func Page(v View, thisMac page.Context) page.Page {
	// invariant: SPEC §6.2, the CLI half of §6.4: the last values and rates,
	// "since boot" in words, the top processes; a stopped VM has none. Any
	// machine but this one starts with `On <it>:` (SPEC §8.1).
	p := page.Page{ID: "activity", Command: "activity", Context: thisMac}
	if !v.ThisMac {
		p.Sections = append(p.Sections, page.OnMachine(v.Label))
	}
	if v.Stopped {
		r := page.Row{ID: "stopped", Label: v.Label + " is stopped", Mark: page.Absent, Reason: page.Stopped}
		p.Sections = append(p.Sections, page.Section{ID: "activity", Title: "Activity", Uncounted: true, Columns: columns()[:1], Items: []page.Item{{Row: &r}}})
		return page.Seal(p)
	}
	banner := page.Row{ID: "machine", Label: v.Label, Mark: page.OK, Cells: []page.Value{page.Texts(head(v)...)}}
	if v.Report.Silent {
		banner.Mark, banner.Reason, banner.Detail = page.Broken, page.Late, page.Text("no answer in 2 s")
	}
	p.Sections = append(p.Sections, page.Section{ID: "machine", Title: "Activity", Role: page.Banner, Uncounted: true, Columns: columns(), Items: []page.Item{{Row: &banner}}})
	if needs := Needs(v); len(needs) > 0 {
		p.Sections = append(p.Sections, page.Section{ID: "needs", Title: "Needs you", Uncounted: true, Columns: columns()[:1], Items: needs})
	}
	p.Sections = append(p.Sections, page.Section{ID: "values", Title: "Now", Uncounted: true, Columns: listing(), Items: values(v)})
	if top := busiest(v); len(top) > 0 {
		p.Sections = append(p.Sections, page.Section{ID: "top", Title: "Top processes", Uncounted: true, Columns: listing(), Items: top})
	}
	if v.Graph > 0 {
		p.Sections = append(p.Sections, graphs(v)...)
	}
	if v.ThisMac {
		p.Hints = append(p.Hints, page.Hint{Command: "station activity --all-machines", Text: "every machine"})
	}
	return page.Seal(p)
}

func Needs(v View) []page.Item {
	out := make([]page.Item, 0, len(v.Report.Findings))
	for i, f := range seen(v) {
		mark, reason := page.Attention, page.Degraded
		if f.Severity == "broken" {
			mark, reason = page.Broken, page.Failing
		}
		r := page.Row{ID: "finding-" + strconv.Itoa(i), Label: f.Text, Mark: mark, Reason: reason}
		if f.Fix != "" {
			r.Hint = page.Cmd(f.Fix)
		}
		out = append(out, page.Item{Row: &r})
	}
	return out
}

type seenFinding struct{ Severity, Section, Subject, Text, Fix string }

func seen(v View) []seenFinding {
	// invariant: a finding of any machine but This Mac names it, and its
	// station fix acts on it: read from This Mac, a bare `station disk` would
	// act on This Mac.
	out := make([]seenFinding, 0, len(v.Report.Findings))
	for _, f := range v.Report.Findings {
		n := seenFinding{Severity: string(f.Severity), Section: string(f.Section), Subject: f.Subject, Text: f.Text, Fix: f.Fix.Command}
		if v.Kind != computer || !v.ThisMac {
			if !strings.HasPrefix(n.Text, v.Name+" ") {
				n.Text = v.Name + ": " + lowerFirst(n.Text)
			}
			if strings.HasPrefix(n.Fix, "station ") && !strings.Contains(n.Fix, " --on ") {
				n.Fix += " --on " + v.Name
			}
		}
		out = append(out, n)
	}
	return out
}

func lowerFirst(s string) string {
	for i, c := range s {
		return string(unicode.ToLower(c)) + s[i+len(string(c)):]
	}
	return s
}

func values(v View) []page.Item {
	r := v.Report
	s := r.Last.Sample
	if r.Silent {
		return []page.Item{
			cell("cpu", "CPU", silent()), cell("memory", "Memory", silent()),
			cell("disk", "Disk", silent()), cell("network", "Network", silent()),
		}
	}
	if run := r.Last.Run; run != nil {
		return []page.Item{
			row("cpu", "CPU", page.Unjudged, activity.Percent(run.CPU)+" of "+v.On),
			row("memory", "Memory", page.Unjudged, activity.Memory(run.Memory)+" resident"),
		}
	}
	rated := r.Window > 0 && counted(s)
	rate := func(perSecond float64) string {
		if !rated {
			return "?"
		}
		return activity.Rate(perSecond)
	}
	cpu := "?"
	if rated {
		cpu = activity.Percent(r.Rates.CPU)
	}
	out := []page.Item{
		row("cpu", "CPU", page.Unjudged, cpu+" of "+strconv.Itoa(s.CPUs)+" CPU", "load "+activity.Load(s.Load)),
		row("memory", "Memory", page.Unjudged, memory(s), "pressure "+pressure(s.Pressure), "swap "+activity.Memory(s.SwapUsed)),
	}
	if s.DiskTotal > 0 {
		out = append(out, row("disk", "Disk", page.Unjudged, activity.Bytes(s.DiskFree)+" free of "+activity.Bytes(s.DiskTotal)))
	}
	out = append(out,
		row("disk-read", "Disk read", page.Unjudged, rate(r.Rates.DiskRead), activity.Bytes(s.DiskRead)+" since boot"),
		row("disk-write", "Disk write", page.Unjudged, rate(r.Rates.DiskWrite), activity.Bytes(s.DiskWritten)+" since boot"),
		row("network", "Network", page.Unjudged, "in "+rate(r.Rates.NetIn), "out "+rate(r.Rates.NetOut)),
		row("network-total", "Network total", page.Unjudged, "received "+activity.Bytes(s.NetIn), "sent "+activity.Bytes(s.NetOut)+" since boot"),
	)
	if s.Thermal != activity.ThermalUnknown && v.Kind == computer {
		mark := page.OK
		switch {
		case s.Thermal >= activity.ThermalCritical:
			mark = page.Broken
		case s.Thermal == activity.ThermalSerious:
			mark = page.Attention
		}
		t := page.Row{ID: "thermal", Label: "Thermal", Mark: mark, Cells: []page.Value{page.Text(activity.Thermal(s.Thermal))}}
		if mark != page.OK {
			t.Reason = page.Degraded
		}
		out = append(out, page.Item{Row: &t})
	}
	if b := s.Battery; b != nil && v.Kind == computer {
		words := []string{strconv.Itoa(b.Percent) + " %", activity.BatteryState(*b)}
		if b.Health > 0 {
			words = append(words, "health "+strconv.Itoa(b.Health)+" %")
		}
		if b.Cycles > 0 {
			words = append(words, strconv.Itoa(b.Cycles)+" cycles")
		}
		out = append(out, row("battery", "Battery", page.Unjudged, words...))
	}
	return out
}

// value: a sample whose kernel counted nothing (no CPU ticks, no memory)
// was not read: its values are unknown, never zeros.
func counted(s activity.Sample) bool { return s.CPUTotal > 0 || s.MemTotal > 0 }

func memory(s activity.Sample) string {
	if s.MemTotal <= 0 {
		return "?"
	}
	return strings.TrimSuffix(activity.Memory(s.MemUsed), " GB") + " of " + activity.Memory(s.MemTotal)
}

func pressure(p activity.Pressure) string {
	if p == "" {
		return "?"
	}
	return string(p)
}

func named(p activity.Process) string {
	if p.VM != "" {
		return p.Name + " (" + p.VM + ")"
	}
	return p.Name
}

func busiest(v View) []page.Item {
	t := v.Report.Last.Top
	var out []page.Item
	if len(t.CPU) > 0 {
		words := make([]string, 0, len(t.CPU))
		for _, p := range t.CPU {
			words = append(words, named(p)+" "+activity.Percent(p.Value))
		}
		out = append(out, row("by-cpu", "Top by CPU", page.Unjudged, words...))
	}
	if len(t.Memory) > 0 {
		words := make([]string, 0, len(t.Memory))
		for _, p := range t.Memory {
			words = append(words, named(p)+" "+activity.Memory(int64(p.Value)))
		}
		out = append(out, row("by-memory", "Top by memory", page.Unjudged, words...))
	}
	return out
}
