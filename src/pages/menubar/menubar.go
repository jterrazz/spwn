package menubar

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
	pageengine "github.com/jterrazz/studio-station/src/pages/engine"
	pagehome "github.com/jterrazz/studio-station/src/pages/home"
	pagekeepawake "github.com/jterrazz/studio-station/src/pages/keepawake"
	"github.com/jterrazz/studio-station/src/ports"
)

const ID page.ID = "menubar"

const (
	ActionNewVM        = "new-vm"
	ActionOpenStation  = "open-station"
	ActionStopAll      = "stop-all"
	ActionStart        = "start"
	ActionKeepAwake    = "awake-keep"
	ActionAwakeOff     = "awake-off"
	ActionUpdateAll    = "update-all"
	ActionOpenAccounts = "open-accounts"

	SectionNeeds    = "needs-you"
	SectionMachines = "machines"
	SectionAwake    = "awake"
	SectionAccounts = "accounts"
)

type Reading struct {
	// value: a nil part is still being read and draws pending.
	Home    *overview.Summary
	Samples []activity.Reading
	All     *model.AllMachines
	Ports   *ports.AllMachines
	Awake   *awake.Report
	// value: every other Mac's Awake, as its engine reports it.
	Macs []awake.Report
	// value: every other Mac's Overview, as its engine reports it.
	Away []overview.Summary
}

func Page(r Reading, thisMac page.Context) page.Page {
	p := page.Page{ID: ID, Command: "menubar", Context: thisMac, Strip: strip(r.Samples, thisMac.Machine)}
	needs := needsYou(r)
	tree := Tree(r.All, r.Ports, homeLine(r))
	markMacs(&tree, r)
	p.Sections = []page.Section{needs, tree, awakeTile(r), accountsTile(r.Home)}
	p.Actions = append([]page.Action{newVM(), openStation(), stopAll(r)}, awakeAction(r.Awake), openAccounts())
	p.Actions = append(p.Actions, updateAll(needs, r.Home)...)
	p.Badge = badge(r, needs)
	p.Live = liveKeys(p)
	return page.Seal(p)
}

// invariant: the icon aggregates and never judges: the count is Needs you's
// own, and a server is down by the machines model's rule; nil while Needs
// you is still read.
func badge(r Reading, needs page.Section) *page.Badge {
	if r.Home == nil {
		return nil
	}
	b := page.Badge{}
	for _, t := range needs.Counts {
		if t.Mark != page.OK && t.Mark != page.Awaiting {
			b.Needs += t.N
		}
	}
	if r.All != nil {
		for _, h := range r.All.Macs {
			b.Down = b.Down || slices.ContainsFunc(h.VMs, model.Down)
		}
	}
	return &b
}

func Stopped(thisMac page.Context, installed bool) page.Page {
	// invariant: the page the app draws when no daemon answers is bundled into
	// it, so it holds only what is true without one.
	row := page.Row{ID: "engine", Label: "engine", Mark: page.Attention, Reason: page.Stopped, Cells: []page.Value{page.Text("stopped")}}
	actions := []page.Action{start(), openStation()}
	if !installed {
		row.Cells = []page.Value{page.Text("not installed")}
		row.Hint = page.Cmd("station setup")
		actions = actions[1:]
	}
	caption := "The engine is stopped"
	if !installed {
		caption = "station is not set up on this Mac"
	}
	s := page.Section{ID: SectionNeeds, Title: "Needs you", Role: page.Actions, Caption: []page.Value{page.Text(caption)}, Columns: columns(), Items: []page.Item{{Row: &row}}}
	return page.Seal(page.Page{ID: ID, Command: "menubar", Context: thisMac, Sections: []page.Section{s}, Actions: actions})
}

func Outdated(thisMac page.Context) page.Page {
	row := page.Row{ID: "engine", Label: "engine", Mark: page.Attention, Reason: page.Outdated, Cells: []page.Value{page.Text("older than this app")}, Hint: page.Cmd("station setup")}
	s := page.Section{ID: SectionNeeds, Title: "Needs you", Role: page.Actions, Caption: []page.Value{page.Text("Update station here (scripts/install.sh)")}, Columns: columns(), Items: []page.Item{{Row: &row}}}
	return page.Seal(page.Page{ID: ID, Command: "menubar", Context: thisMac, Sections: []page.Section{s}, Actions: []page.Action{openStation()}})
}

func columns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}
}

// value: the three measures of This Mac, CPU, Memory and Network, each
// with its sparkline over the readings kept, each read live under its key.
func strip(readings []activity.Reading, thisMac string) *page.Strip {
	cpu, memory, net := LiveKey(LiveCPU, thisMac), LiveKey(LiveMemory, thisMac), LiveKey(LiveNet, thisMac)
	if len(readings) < 2 {
		return &page.Strip{CPU: live(page.Pending(), cpu), Memory: live(page.Pending(), memory)}
	}
	m := measured(readings)
	s := page.Strip{CPU: live(m.cpu, cpu), Spark: m.cpus, MemorySpark: m.memories, NetSpark: m.nets, Net: []page.Value{live(m.net, net)}}
	if m.memory.Kind != "" {
		s.Memory = live(m.memory, memory)
	}
	return &s
}

func live(v page.Value, key string) page.Value {
	v.Live = key
	return v
}

// value: the live keys the values of a page follow, each once, in order.
func liveKeys(p page.Page) []string {
	var keys []string
	add := func(v page.Value) {
		if v.Live != "" && !slices.Contains(keys, v.Live) {
			keys = append(keys, v.Live)
		}
	}
	if s := p.Strip; s != nil {
		add(s.CPU)
		add(s.Memory)
		for _, v := range s.Net {
			add(v)
		}
	}
	for _, sec := range p.Sections {
		for _, it := range sec.Items {
			if it.Row != nil {
				for _, v := range it.Row.Cells {
					add(v)
				}
			}
		}
	}
	return keys
}

// invariant: Needs you holds every finding, the line vision panel a draws
// naming each and unfolding into them; the section keeps the full counts
// (Uncounted) and its caption says them all.
func needsYou(r Reading) page.Section {
	s := page.Section{ID: SectionNeeds, Title: "Needs you", Role: page.Actions, Uncounted: true, Columns: columns()}
	if r.Home == nil {
		s.Uncounted = false
		s.Items = []page.Item{{Row: &page.Row{ID: "reading", Label: "reading", Mark: page.Awaiting}}}
		return s
	}
	var rows []page.Item
	counts := map[page.Mark]int{}
	for _, body := range pagehome.Body(*r.Home, true) {
		if body.ID != SectionNeeds {
			continue
		}
		for _, it := range body.Items {
			if it.Row == nil {
				continue
			}
			if n := len(rows); n < len(r.Home.Needs) && r.Home.Needs[n].Section == section.Background {
				row := *it.Row
				row.Label, row.Detail = taskWords(r.Home.Needs[n]), page.Text(r.Home.Needs[n].Subject)
				it.Row = &row
			}
			counts[it.Row.Mark]++
			rows = append(rows, it)
		}
	}
	for _, m := range page.Marks() {
		if counts[m] > 0 {
			s.Counts = append(s.Counts, page.Tally{Mark: m, N: counts[m]})
		}
	}
	s.Caption = []page.Value{page.Text(things(len(rows)))}
	s.Glance = glance(r.Home.Needs)
	s.Items = rows
	if len(rows) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "Nothing needs you"}}}
	}
	return s
}

// value: Needs you in the vision's short words, one part per kind of
// finding in the one order: the updates summed, an account's sign-in, a
// folder to move, a setting, a failing or left-over task; any other
// finding by its words up to a comma.
func glance(needs []overview.Need) []page.Value {
	var updates, failing, left int
	var accountsOut, folders, settings, others []string
	for _, n := range needs {
		switch n.Section {
		case section.Apps:
			updates += leading(n.Text)
		case section.Accounts:
			accountsOut = append(accountsOut, n.Subject+" sign-in")
		case section.Disk:
			folders = append(folders, n.Subject)
		case section.Settings:
			settings = append(settings, strings.ToLower(strings.ReplaceAll(n.Text, ":", "")))
		case section.Background:
			if n.Broken {
				failing++
			} else {
				left++
			}
		default:
			head, _, _ := strings.Cut(n.Text, ",")
			others = append(others, head)
		}
	}
	var parts []string
	if updates > 0 {
		parts = append(parts, counted(updates, "update", "updates"))
	}
	parts = append(parts, accountsOut...)
	parts = append(parts, grouped(folders, "folders to move")...)
	parts = append(parts, grouped(settings, "settings")...)
	if failing > 0 {
		parts = append(parts, counted(failing, "failing task", "failing tasks"))
	}
	if left > 0 {
		parts = append(parts, counted(left, "left-over task", "left-over tasks"))
	}
	parts = append(parts, others...)
	out := make([]page.Value, 0, len(parts))
	for _, p := range parts {
		out = append(out, page.Text(p))
	}
	return out
}

// value: a background task's line in plain words, by the app it belongs
// to or by its name, never its launchd label, which stays the row's
// detail.
func taskWords(n overview.Need) string {
	name := taskName(n.Subject)
	if owner, known := catalog.LaunchOwnerOf(n.Subject); known {
		name = strings.TrimSuffix(owner.App, ".app")
		if !n.Broken {
			return name + " left a helper behind"
		}
	}
	if n.Broken {
		return "A background task is failing: " + name
	}
	return "A background task is left over: " + name
}

// value: a launchd label's own words: its maker's domain dropped, its
// parts split, a lone letter left out (`com.jterrazz.brain.x-snapshot`
// reads `brain snapshot`).
func taskName(label string) string {
	parts := strings.Split(label, ".")
	if len(parts) > 2 {
		parts = parts[2:]
	}
	var words []string
	for _, w := range strings.FieldsFunc(strings.Join(parts, " "), func(r rune) bool { return r == ' ' || r == '-' || r == '_' }) {
		if len(w) > 1 {
			words = append(words, w)
		}
	}
	if len(words) == 0 {
		return label
	}
	return strings.Join(words, " ")
}

func leading(text string) int {
	head, _, _ := strings.Cut(text, " ")
	if n, err := strconv.Atoi(head); err == nil && n > 0 {
		return n
	}
	return 1
}

func counted(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

func grouped(words []string, many string) []string {
	if len(words) > 1 {
		return []string{strconv.Itoa(len(words)) + " " + many}
	}
	return words
}

func things(n int) string {
	switch n {
	case 0:
		return "nothing needs you"
	case 1:
		return "1 thing needs you"
	}
	return strconv.Itoa(n) + " things need you"
}

// invariant: MB-16: a Needs you line whose fix is the apps' update gets
// its own Update all, which runs that call on This Mac for what the line
// counts: every update of its manager (`brew`), or the one app it names;
// a line that names neither updates everything. What fails stays in Needs
// you with its fix.
func updateAll(needs page.Section, home *overview.Summary) []page.Action {
	if home == nil {
		return nil
	}
	var out []page.Action
	for i, n := range home.Needs {
		id := strconv.Itoa(i) + "-" + n.Subject
		if n.Call != CallUpdate || !slices.ContainsFunc(needs.Items, func(it page.Item) bool { return it.Row != nil && it.Row.ID == id }) {
			continue
		}
		run := &page.Run{Call: CallUpdate}
		switch {
		case n.Args["method"] != "":
			run.In = map[string]string{"Method": `["` + n.Args["method"] + `"]`}
		case n.Args["name"] != "":
			run.In = map[string]string{"Names": `["` + n.Args["name"] + `"]`}
		}
		out = append(out, page.Action{ID: ActionUpdateAll + ":" + id, Label: "Update all", On: page.OnRow, Rows: []string{id}, Priority: 5, Mutates: true, Run: run})
	}
	return out
}

// value: the line under a Mac in Home: its Awake; another Mac's then says
// how many things need you there (vision panel a: `never sleeps · 1 needs
// you`).
func homeLine(r Reading) func(h model.MacView) page.Value {
	return func(h model.MacView) page.Value {
		var parts []page.Value
		switch {
		case h.ThisMac && r.Awake != nil:
			if w := awakeWords(*r.Awake); w != "" {
				parts = append(parts, page.Text("awake "+w))
			}
		case !h.ThisMac:
			for _, m := range r.Macs {
				if m.Mac == h.Name && m.NeverSleeps {
					parts = append(parts, page.Text("never sleeps"))
				}
			}
			if s := away(r, h.Name); s != nil && len(s.Needs) > 0 {
				parts = append(parts, page.Text(needWords(len(s.Needs))))
			}
		}
		if len(parts) == 0 {
			return page.None()
		}
		return page.List(parts...)
	}
}

func needWords(n int) string {
	if n == 1 {
		return "1 needs you"
	}
	return strconv.Itoa(n) + " need you"
}

func away(r Reading, mac string) *overview.Summary {
	for i, s := range r.Away {
		if s.Head.Name == mac {
			return &r.Away[i]
		}
	}
	return nil
}

// invariant: a Mac's dot in Home says what needs you there, This Mac's
// from its own Needs you, another's from its engine's: amber for
// anything, red when one of its ports is reachable outside Tailscale
// undeclared (vision panels a and g); a Mac that does not answer keeps its
// own mark.
func markMacs(s *page.Section, r Reading) {
	if r.All == nil {
		return
	}
	for _, it := range s.Items {
		if it.Group == nil || it.Group.Head.Mark != page.OK {
			continue
		}
		head := &it.Group.Head
		for _, h := range r.All.Macs {
			if h.Name != head.ID {
				continue
			}
			switch {
			case h.ThisMac && r.Home != nil:
				head.Mark, head.Reason = needsMark(r.Home.Needs)
			case !h.ThisMac:
				if a := away(r, h.Name); a != nil {
					head.Mark, head.Reason = needsMark(a.Needs)
				}
			}
		}
	}
}

func needsMark(needs []overview.Need) (page.Mark, page.Reason) {
	mark, reason := page.OK, page.Reason("")
	for _, n := range needs {
		if n.Section == section.Ports && n.Broken {
			return page.Broken, page.Exposed
		}
		mark, reason = page.Attention, page.Degraded
	}
	return mark, reason
}

func awakeWords(r awake.Report) string {
	switch {
	case r.Session != nil:
		zone := r.Zone
		if zone == nil {
			zone = time.UTC
		}
		return "until " + r.Session.Until.In(zone).Format("15:04")
	case r.NeverSleeps:
		return "never sleeps"
	}
	return ""
}

// value: the Awake tile: This Mac's keep-awake first, then each other Mac's
// mode; its switch is This Mac's.
func awakeTile(r Reading) page.Section {
	var parts []page.Value
	self := ""
	if r.Awake != nil {
		self = r.Awake.Mac
		parts = append(parts, page.Text(page.ThisMac+" "+cmpWord(awakeWords(*r.Awake), "sleeps")))
	}
	for _, m := range r.Macs {
		if !m.ThisMac {
			parts = append(parts, page.Text(macLabel(r, m.Mac, self)+" "+cmpWord(awakeWords(m), "sleeps")))
		}
	}
	value := page.Pending()
	if r.Awake != nil {
		value = page.List(parts...)
	}
	row := page.Row{ID: pagekeepawake.RowAwake, Label: "Awake", Cells: []page.Value{value}}
	return page.Section{ID: SectionAwake, Title: "Awake", Role: page.Summary, Columns: columns(), Items: []page.Item{{Row: &row}}}
}

// value: a Mac as the Machines list names it (`mac-mini`), else by the
// interface's rule for a Mac's name.
func macLabel(r Reading, mac, self string) string {
	if r.All != nil {
		for _, h := range r.All.Macs {
			if h.Name == mac {
				return model.Label(*r.All, h)
			}
		}
	}
	return page.MacName(mac, self)
}

func cmpWord(s, otherwise string) string {
	if s == "" {
		return otherwise
	}
	return s
}

// value: the Accounts tile: This Mac's Accounts line of its Overview.
func accountsTile(home *overview.Summary) page.Section {
	row := page.Row{ID: "accounts", Label: "Accounts", Cells: []page.Value{page.Pending()}}
	if home != nil {
		for _, body := range pagehome.Body(*home, true) {
			for _, it := range body.Items {
				if body.ID == "sections" && it.Row != nil && it.Row.ID == "accounts" {
					row.Cells = []page.Value{signIns(it.Row.Cells[0])}
				}
			}
		}
	}
	return page.Section{ID: SectionAccounts, Title: "Accounts", Role: page.Summary, Columns: columns(), Items: []page.Item{{Row: &row}}}
}

// value: the Accounts tile's words: how many are signed in, then each
// account whose sign-in failed said so (`npm needs sign-in`).
func signIns(v page.Value) page.Value {
	if v.Kind != page.KindList {
		return v
	}
	out := make([]page.Value, 0, len(v.Items))
	for _, it := range v.Items {
		if it.Kind == page.KindText && (it.Label == string(page.Broken) || it.Label == string(page.Attention)) {
			it.Text += " needs sign-in"
		}
		out = append(out, it)
	}
	v.Items = out
	return v
}

// value: Accounts opens a terminal on the accounts of This Mac.
func openAccounts() page.Action {
	return page.Action{ID: ActionOpenAccounts, Label: "Accounts", On: page.OnRow, Rows: []string{SectionAccounts}, Priority: 6, Run: &page.Run{Terminal: true, Argv: []string{"accounts"}}}
}

// value: the calls the menu bar's actions make, by their calls.json names.
const (
	CallCreate    = "machines.create-workspace"
	CallStop      = "engine.stop"
	CallKeepAwake = "awake.keep"
	CallAwakeOff  = "awake.off"
	CallUpdate    = "apps.update"
	CallAddMac    = "machines.add"
	CallStopVM    = "machines.stop"
	CallStartVM   = "machines.start"
	CallDeleteVM  = "machines.delete"
	CallRestartVM = "machines.restart"
	CallSharePort = "ports.share"
	CallTakeBack  = "accounts.take-back"
)

func gigabytes(n int64) string { return strconv.FormatInt(n<<30, 10) }

// value: New VM…'s sheet: the create form of the CLI and the TUI, its
// questions in the same words.
func newVM() page.Action {
	return page.Action{ID: ActionNewVM, Key: "cmd+n", Label: "New VM", On: page.OnPage, Priority: 1, Mutates: true,
		Run: &page.Run{Call: CallCreate, In: map[string]string{"Request.Image": "linux", "Request.Network": "nat"}},
		Form: &page.Form{Title: "New VM", Fields: []page.Field{
			{Key: "Request.Lifetime", Label: "Will it last?", Kind: page.FieldSelect, Default: "throwaway", Options: []page.Option{
				{Value: "throwaway", Label: "Throwaway", Note: "Deleted when it powers off from inside."},
				{Value: "permanent", Label: "Permanent", Note: "Kept until you delete it."},
			}},
			{Key: "Request.Size.MemoryBytes", Label: "Memory", Kind: page.FieldSelect, Default: gigabytes(4), Options: []page.Option{{Value: gigabytes(4), Label: "4 GB"}, {Value: gigabytes(8), Label: "8 GB"}}},
		}}}
}

// value: Open in Terminal: a new terminal runs station, on the machine
// these words name; the one way the menu bar starts a program.
func openIn(words ...string) page.Action {
	return page.Action{ID: ActionOpenStation, Key: "cmd+t", Label: "Open in Terminal", On: page.OnPage, Priority: 2, Run: &page.Run{Terminal: true, Argv: words}}
}

func openStation() page.Action { return openIn() }

func start() page.Action {
	return page.Action{ID: ActionStart, Label: "Start the engine", On: page.OnPage, Priority: 1, Mutates: true, Run: &page.Run{Terminal: true, Argv: []string{"engine", "start"}}}
}

func stopAll(r Reading) page.Action {
	var w engine.Work
	if r.All != nil {
		for _, h := range r.All.Macs {
			if !h.ThisMac {
				continue
			}
			for _, vm := range h.VMs {
				switch {
				case vm.State != model.Running:
				case vm.Lifetime == model.AlwaysOn:
					w.Servers = append(w.Servers, vm.Name)
				default:
					w.VMs = append(w.VMs, engine.Stopped{Name: vm.Name, Lifetime: string(vm.Lifetime), Result: engine.ResultStopped})
				}
			}
		}
	}
	c := pageengine.StopConfirm(w, page.ThisMac)
	return page.Action{ID: ActionStopAll, Label: "Stop everything on " + page.ThisMac, On: page.OnPage, Priority: 3, Mutates: true, Confirm: &c,
		Run: &page.Run{Call: CallStop, In: map[string]string{"Confirm": yes, "By": "the menu bar"}}}
}

// invariant: the Awake switch is the tile's action, awake with its form
// while off and sleep while a session holds.
func awakeAction(r *awake.Report) page.Action {
	rows := []string{pagekeepawake.RowAwake}
	if r != nil && r.Session != nil && r.Engine == awake.EngineRunning {
		return page.Action{ID: ActionAwakeOff, Label: "sleep", On: page.OnRow, Rows: rows, Priority: 4, Mutates: true, Run: &page.Run{Call: CallAwakeOff}}
	}
	terms := []int{1, 4, 8}
	opts := make([]page.Option, 0, len(terms))
	for _, h := range terms {
		opts = append(opts, page.Option{Value: strconv.FormatInt(int64(time.Duration(h)*time.Hour), 10), Label: strconv.Itoa(h) + " h"})
	}
	return page.Action{ID: ActionKeepAwake, Label: "awake", On: page.OnRow, Rows: rows, Priority: 4, Mutates: true, Run: &page.Run{Call: CallKeepAwake},
		Form: &page.Form{Title: "Awake", Fields: []page.Field{
			{Key: "For", Label: "Keep awake for", Kind: page.FieldSelect, Default: strconv.FormatInt(int64(4*time.Hour), 10), Options: opts},
			{Key: "Lid", Label: "Lid", Kind: page.FieldSelect, Default: "Lid=false", Options: []page.Option{
				{Value: "Lid=false", Label: "open only"}, {Value: "Lid=true", Label: "closed"}, {Value: "Lid=true Battery=true", Label: "closed on battery"},
			}},
		}}}
}

type Levels func(gesture string, kind machine.Kind) string

// value: the levels a gesture on a machine of a kind asks at, as the
// call's confirmation rule gives them (confirm.For, its caller's): a page
// only draws them.
const (
	LevelNone     = "none"
	LevelYesNo    = "yes-no"
	LevelTypeName = "type-name"
	yes           = "yes"
)

// value: the question a gesture on a machine asks, at the level the call
// asks it, never the surface's: none, yes or no, or the name to type.
func ask(levels Levels, gesture string, k machine.Kind, name, question, verb string, lines ...string) *page.Confirm {
	values := make([]page.Value, 0, len(lines)+1)
	for _, l := range lines {
		values = append(values, page.Text(l))
	}
	switch levels(gesture, k) {
	case LevelNone:
		return nil
	case LevelTypeName:
		values = append(values, page.Text("Type "+name+" to "+verb+" it."))
		return &page.Confirm{Question: question, Lines: values, Danger: true, Type: name, Yes: titled(verb)}
	}
	return &page.Confirm{Question: question, Lines: values, Yes: titled(verb)}
}

// value: the call's input that answers the question ask drew: yes for a
// yes or no, nothing for a typed one, whose typed name the sheet adds.
func answered(c *page.Confirm, in map[string]string) map[string]string {
	if c != nil && c.Type == "" {
		in["Confirm"] = yes
	}
	return in
}

func titled(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func Opening(id page.ID, thisMac page.Context) page.Page {
	// invariant: the frame of one of the menu bar's own pages, before its
	// first read.
	return page.Page{ID: id, Context: thisMac, State: page.Loading}
}

func Unread(id page.ID, thisMac page.Context, err error) page.Page {
	return page.Seal(page.Page{ID: id, Context: thisMac, Error: &page.Error{Message: err.Error()}})
}
