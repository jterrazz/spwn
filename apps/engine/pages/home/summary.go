package home

import (
	"sort"
	"strconv"
	"strings"
	"time"

	coresection "github.com/jterrazz/studio-station/apps/engine/core/section"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
)

// value: the Needs-you lines an Overview shows folded; --all shows every one.
const folded = 5

func Summary(s overview.Summary, thisMac page.Context, all bool) page.Page {
	// invariant: SPEC §5.1: the first line sums the machine up, Needs you
	// comes first, then one line per section in the one order, then the VMs
	// of a Mac, then one or two next steps.
	sections := append([]page.Section{{ID: "overview", Title: title(s, thisMac), Caption: caption(s, thisMac), Joined: true, Role: page.Banner, Uncounted: true}}, Body(s, all)...)
	state := page.Ready
	if s.Pending {
		state = page.Loading
	}
	p := page.Page{ID: "overview", Command: "overview", Context: thisMac, State: state, Sections: sections, Hints: Hints(s, all)}
	if len(s.Needs) > 0 {
		p.Actions = []page.Action{{Key: KeyFix, Label: "fix", On: page.OnRow, Rows: needsRows(sections), Priority: 1, Mutates: true}}
	}
	if s.Engine == overview.EngineStopped {
		p.Actions = append(p.Actions, page.Action{Key: KeyStartEngine, Label: "start the engine", On: page.OnPage, Priority: 2, Mutates: true, ID: ActionStartEngine, Run: &page.Run{Call: CallStartEngine}})
	}
	return page.Seal(p)
}

func needsRows(sections []page.Section) []string {
	var rows []string
	for _, s := range sections {
		if s.ID != "needs-you" {
			continue
		}
		for _, it := range s.Items {
			if it.Row != nil {
				rows = append(rows, it.Row.ID)
			}
		}
	}
	return rows
}

func Body(s overview.Summary, all bool) []page.Section {
	// value: an Overview below its first line: Needs you, then one line per
	// section, then the VMs of a Mac; a server says what station leaves to you.
	var sections []page.Section
	if len(s.Needs) > 0 {
		sections = append(sections, needsYou(s.Needs, all))
	}
	lines := make([]page.Item, 0, len(s.Lines)+2)
	for _, l := range s.Lines {
		row := page.Row{ID: string(l.Section), Label: sectionTitle(string(l.Section)), Cells: []page.Value{lineValue(l)}}
		if l.Pending {
			row.Mark = page.Awaiting
		}
		lines = append(lines, page.Item{Row: &row})
	}
	if s.Head.ThisMac && len(s.VMs) > 0 {
		row := page.Row{ID: "vms", Label: "VMs", Cells: []page.Value{vmsValue(s.VMs)}}
		lines = append(lines, page.Item{Row: &row})
	}
	if s.Head.Server() {
		lines = append(lines, page.Item{Note: &page.Note{Text: "Apps, settings, accounts: not managed by station"}})
	}
	sections = append(sections, page.Section{ID: "sections", Role: page.Banner, Uncounted: true, Columns: []page.Column{
		{Key: "section", Kind: page.KindListing, Keep: 100}, {Key: "line", Kind: page.KindList, Keep: 90},
	}, Items: lines})
	if len(s.Readiness) > 0 {
		sections = append(sections, section("readiness", "Mac readiness", checkRows(s.Readiness)))
	}
	return sections
}

// value: the Mac this binary runs on; another Mac read through --on is
// itself "here" on its side and goes by its name on this one.
func thisMac(s overview.Summary, thisMac page.Context) bool {
	return s.Head.ThisMac && s.Head.Mac() && (!thisMac.Registered || thisMac.Machine == "" || thisMac.Machine == s.Head.Name)
}

func title(s overview.Summary, thisMachine page.Context) string {
	if thisMac(s, thisMachine) {
		return "This Mac"
	}
	if s.Head.Mac() && s.Head.Name != thisMachine.Machine {
		return page.MacName(s.Head.Name, thisMachine.Machine)
	}
	return s.Head.Name
}

func caption(s overview.Summary, thisMachine page.Context) []page.Value {
	if !s.Head.Mac() {
		// invariant: a VM read inside it names its kind and how long it has
		// run; which Mac it is on and its lifetime are its Mac's to tell.
		parts := []page.Value{page.Text(kindWord(s.Head))}
		if s.Head.Up > 0 {
			parts = append(parts, Up(s.Head.Up))
		}
		return parts
	}
	parts := []page.Value{page.Text(s.Head.Name)}
	if !thisMac(s, thisMachine) {
		parts = []page.Value{page.Text(kindWord(s.Head))}
	}
	switch len(s.Needs) {
	case 0:
		parts = append(parts, page.Text("nothing needs you"))
	case 1:
		parts = append(parts, page.Text("1 thing needs you"))
	default:
		parts = append(parts, page.Text(strconv.Itoa(len(s.Needs))+" things need you"))
	}
	switch s.Engine {
	case overview.EngineRunning:
		parts = append(parts, page.Text("Engine running"))
	case overview.EngineStopped:
		parts = append(parts, page.Text("Engine stopped"))
	case overview.EngineNone:
	}
	return parts
}

func needsYou(needs []overview.Need, all bool) page.Section {
	items := make([]page.Item, 0, len(needs))
	for i, n := range needs {
		if !all && i == folded {
			items = append(items, page.Item{Note: &page.Note{Text: strconv.Itoa(len(needs)-folded) + " more: station overview --all"}})
			break
		}
		mark, reason := page.Attention, page.Degraded
		if n.Broken {
			mark, reason = page.Broken, page.Failing
		}
		row := page.Row{ID: strconv.Itoa(i) + "-" + n.Subject, Label: n.Text, Mark: mark, Reason: reason, Cells: []page.Value{fixValue(n)}}
		items = append(items, page.Item{Row: &row})
	}
	columns := []page.Column{{Key: "finding", Kind: page.KindText, Keep: 100}, {Key: "fix", Kind: page.KindCommand, Keep: 50}}
	return page.Section{ID: "needs-you", Title: "Needs you", Uncounted: true, Columns: columns, Items: items}
}

func fixValue(n overview.Need) page.Value {
	switch {
	case n.Call != "":
		return page.Text(fixWords(n.Call, n.Args))
	case n.Command != "":
		return page.Text(n.Command)
	case n.Path != "":
		return page.Text(n.Path)
	}
	return page.None()
}

func fixWords(name string, args map[string]string) string {
	// invariant: a fix's call printed as the command that runs it: its noun
	// and verb, the name or path it acts on, then each other input a flag.
	noun, verb, _ := strings.Cut(name, ".")
	words := []string{"station", noun}
	if verb != "list" && verb != "read" && verb != "run" {
		words = append(words, verb)
	}
	for _, key := range []string{"name", "path"} {
		if v, ok := args[key]; ok {
			words = append(words, v)
		}
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		if k != "name" && k != "path" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		words = append(words, "--"+k, args[k])
	}
	return strings.Join(words, " ")
}

func lineValue(l overview.Line) page.Value {
	switch {
	case l.Pending:
		return page.Pending()
	case l.Unread != "":
		return page.Text(l.Unread)
	case l.Section == coresection.SSH && len(l.Parts) == 1:
		return page.Text(catalog.SSHCommand("", l.Parts[0].Word))
	}
	parts := make([]page.Value, 0, len(l.Parts))
	for _, p := range l.Parts {
		parts = append(parts, partValue(p))
	}
	return page.List(parts...)
}

func partValue(p overview.Part) page.Value {
	var v page.Value
	switch {
	case p.Bytes > 0 || p.Word == "free" || p.Word == "can rebuild":
		v = page.Size(p.Bytes)
		v.Unit = p.Word
	case p.Of > 0:
		v = page.Text(strconv.Itoa(p.N) + " of " + strconv.Itoa(p.Of))
	case p.Counted && p.State == "":
		v = page.Text(p.Word + " " + strconv.Itoa(p.N))
	case p.Counted && p.Word == "":
		v = page.Text(strconv.Itoa(p.N))
	case p.Counted:
		v = page.Count(int64(p.N), p.Word)
	default:
		v = page.Text(p.Word)
	}
	if p.State == "" {
		return v
	}
	if p.State == catalog.StateUnknown {
		return page.Labeled(string(page.NotKnown), v)
	}
	return page.Labeled(string(pages.MarkOf(p.State)), v)
}

func vmsValue(vms []overview.VM) page.Value {
	parts := make([]page.Value, 0, len(vms))
	for _, vm := range vms {
		if vm.Running {
			parts = append(parts, page.Labeled(string(page.OK), page.Text(vm.Name+" running")))
			continue
		}
		parts = append(parts, page.Labeled(string(page.Absent), page.Text(vm.Name+" stopped")))
	}
	return page.List(parts...)
}

func kindWord(h overview.Head) string {
	switch {
	case h.Server():
		return "Server"
	case h.Mac():
		return "Mac"
	}
	return "Workspace"
}

func Up(d time.Duration) page.Value {
	// value: how long a machine has run, in the first line's words ("up 2 h").
	switch {
	case d >= 24*time.Hour:
		return page.Text("up " + strconv.Itoa(int(d/(24*time.Hour))) + " d")
	case d >= time.Hour:
		return page.Text("up " + strconv.Itoa(int(d/time.Hour)) + " h")
	}
	return page.Text("up " + strconv.Itoa(max(int(d/time.Minute), 1)) + " min")
}

func Hints(s overview.Summary, all bool) []page.Hint {
	var out []page.Hint
	for _, n := range s.Needs {
		if n.Call != "" {
			out = append(out, page.Hint{Command: fixWords(n.Call, n.Args)})
			break
		}
	}
	if !all && len(s.Needs) > folded {
		out = append(out, page.Hint{Command: "station overview --all", Text: "for every line"})
	}
	return out
}

func sectionTitle(id string) string {
	switch id {
	case "":
		return ""
	case string(coresection.SSH):
		return "SSH"
	}
	return strings.ToUpper(id[:1]) + id[1:]
}
