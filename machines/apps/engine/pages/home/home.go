package home

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/apps"

	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
	pageapps "github.com/jterrazz/studio-station/apps/engine/pages/apps"
	pageports "github.com/jterrazz/studio-station/apps/engine/pages/ports"
	pagesettings "github.com/jterrazz/studio-station/apps/engine/pages/settings"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

const KeyFix = "enter"

// value: a Mac whose engine is stopped starts it from its Overview (SPEC
// §9.10), with the call the menu bar's Start the engine makes.
const (
	KeyStartEngine    = "s"
	ActionStartEngine = "start-engine"
	CallStartEngine   = "engine.start"
)

// invariant: enter runs an action's fix only when station owns it; an
// External command (launchctl, sudo) is shown and never run (H3).
func fixes(actions page.Section) []string {
	var out []string
	for _, it := range actions.Items {
		if it.Row != nil && it.Row.Hint != nil && !it.Row.Hint.External {
			out = append(out, it.Row.ID)
		}
	}
	return out
}

func Columns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}
}

func Snapshot(s Overview, thisMac page.Context, all bool) page.Page {
	detail := []page.Section{
		section("machine", "Machine", machineRows(s)),
		section("environment", "Environment", environmentRows(s)),
		section("workspace", "Workspace", workspaceRows(s.Workspace)),
		section("identity", "Identity", checkRows(s.Identity)),
	}
	if len(s.Background) > 0 {
		detail = append(detail, backgroundSection(s.Background))
	}
	if len(s.Gateways) > 0 {
		detail = append(detail, section("gateways", "Gateways", checkRows(s.Gateways)))
	}
	detail = append(detail, appsSection(s), settingsSection(s))
	if len(s.Readiness) > 0 {
		detail = append(detail, section("readiness", "Mac readiness", checkRows(s.Readiness)))
	}
	fold := page.Folded
	if all {
		fold = page.Open
	}
	for i := range detail {
		detail[i].Fold = fold
	}
	sealed := page.Seal(page.Page{Sections: detail})
	p := page.Page{
		ID:      "home",
		Command: "status",
		Context: thisMac,
		Hints: []page.Hint{
			{Command: "station overview --all", Text: "for every row"},
			{Command: "station overview | cat", Text: "for records", External: true},
		},
	}
	if all {
		p.Hints = p.Hints[1:]
	}
	actions := todo(sealed.Sections, s.Background)
	p.Sections = append([]page.Section{actions, summary(s, sealed.Sections)}, sealed.Sections...)
	if rows := fixes(actions); len(rows) > 0 {
		p.Actions = []page.Action{{Key: KeyFix, Label: "fix", On: page.OnRow, Rows: rows, Mutates: true}}
	}
	return page.Seal(p)
}

func section(id, title string, rows []page.Row) page.Section {
	items := make([]page.Item, 0, len(rows))
	for i := range rows {
		items = append(items, page.Item{Row: &rows[i]})
	}
	return page.Section{ID: id, Title: title, Columns: Columns(), Items: items}
}

func appsSection(s Overview) page.Section {
	if s.AppsPending {
		return section("apps", "Apps", []page.Row{pending("catalogue", "catalogue")})
	}
	var items []page.Item
	for _, r := range s.Apps {
		if r.State != catalog.StateOK {
			row := pageapps.Row(r)
			items = append(items, page.Item{Row: &row})
		}
	}
	return page.Section{ID: "apps", Title: "Apps", Columns: pageapps.Columns(), Items: items, Uncounted: true, PipeCounts: true, Counts: appCounts(s.Apps)}
}

func settingsSection(s Overview) page.Section {
	if s.SettingsPending {
		return section("settings", "Settings", []page.Row{pending("packages", "packages")})
	}
	var items []page.Item
	for _, r := range s.Settings {
		if r.State != catalog.StateOK {
			row := pagesettings.ReportRow(r)
			items = append(items, page.Item{Row: &row})
		}
	}
	return page.Section{ID: "settings", Title: "Settings", Columns: pagesettings.ReportColumns(), Items: items, Uncounted: true, PipeCounts: true, Counts: settingsCounts(s.Settings)}
}

func tallies(marks []page.Mark) []page.Tally {
	var out []page.Tally
	for _, m := range page.Marks() {
		n := 0
		for _, x := range marks {
			if x == m {
				n++
			}
		}
		if n > 0 {
			out = append(out, page.Tally{Mark: m, N: n})
		}
	}
	return out
}

func appCounts(reports []apps.Report) []page.Tally {
	marks := make([]page.Mark, 0, len(reports))
	for _, r := range reports {
		marks = append(marks, pages.MarkOf(r.State))
	}
	return tallies(marks)
}

func settingsCounts(reports []settings.Report) []page.Tally {
	marks := make([]page.Mark, 0, len(reports))
	for _, r := range reports {
		marks = append(marks, pages.MarkOf(r.State))
	}
	return tallies(marks)
}

func reasonOf(m page.Mark) page.Reason {
	switch m {
	case page.Attention:
		return page.Degraded
	case page.Broken:
		return page.Down
	case page.Absent:
		return page.Missing
	case page.OK, page.Awaiting, page.OffHere, page.NotForThisMachine, page.NotKnown, page.Unjudged:
	}
	return ""
}

func row(id string, mark page.Mark, value page.Value, detail string) page.Row {
	return page.Row{ID: id, Label: strings.ReplaceAll(id, "-", " "), Mark: mark, Reason: reasonOf(mark), Cells: []page.Value{value}, Detail: page.Text(detail)}
}

// invariant: a pending row says only its name: its value is the probe's, and
// the probe has not answered.
func pending(id, label string) page.Row {
	return page.Row{ID: id, Label: label, Mark: page.Awaiting}
}

// value: a protection the owner turned off on purpose is a value, not an absence (M3).
func offByChoice(c overview.Check) bool {
	return c.State == catalog.StateAbsent && c.Value == "off" && slices.Contains([]string{"sip", "remote-login"}, c.Name)
}

func checkRows(checks []overview.Check) []page.Row {
	rows := make([]page.Row, 0, len(checks))
	for _, c := range checks {
		if c.State == catalog.StatePending {
			rows = append(rows, pending(c.Name, strings.ReplaceAll(c.Name, "-", " ")))
			continue
		}
		value := page.Text(c.Value)
		if c.Value == "" && c.State == catalog.StateUnknown {
			value = page.Text("unknown")
		}
		mark := pages.MarkOf(c.State)
		if offByChoice(c) {
			mark, value = page.OK, page.Off()
		}
		rows = append(rows, fixed(row(c.Name, mark, value, c.Detail)))
	}
	return rows
}

// invariant: a domain detail that is a station command is the row's fix, so
// it renders as a hint (`→ station engine start`) and the pipe's last field is unchanged.
func fixed(r page.Row) page.Row {
	if r.Detail.Kind == page.KindText && strings.HasPrefix(r.Detail.Text, "station ") {
		r.Hint, r.Detail = page.Cmd(r.Detail.Text), page.None()
	}
	return r
}

func machineRows(s Overview) []page.Row {
	m := s.Machine
	facts := page.Texts(machineWord(m.Kind), strings.TrimSpace(m.OS+" "+m.Arch), m.User, m.Shell)
	name := page.Row{ID: "name", Label: "name", Mark: page.OK, Cells: []page.Value{page.Text(m.Name)}, Detail: facts}
	if !m.Registered {
		name.Mark, name.Reason, name.Cells = page.Attention, page.Unset, []page.Value{page.Text("unregistered")}
	}
	rows := []page.Row{name}
	if s.System.Pending {
		rows = append(rows, pending("uptime", "uptime"))
	}
	if s.System.Uptime != "" {
		detail := ""
		if s.System.Battery != "" {
			detail = "battery " + s.System.Battery
		}
		rows = append(rows, row("uptime", page.OK, page.Text(s.System.Uptime), detail))
	}
	rows = append(rows, setupRow(s.Setup))
	if d := s.Station.Engine; d.State == catalog.StatePending {
		rows = append(rows, pending("engine", "engine"))
	} else if d.Value != "" {
		rows = append(rows, fixed(row("engine", pages.MarkOf(d.State), page.Text(d.Value), d.Detail)))
	}
	return rows
}

func setupRow(s overview.Setup) page.Row {
	if s.State == catalog.StatePending {
		return pending("setup", "setup")
	}
	fix := page.Cmd("station init")
	for _, step := range s.Steps {
		if step.State == catalog.StateOK {
			continue
		}
		r := page.Row{ID: "setup", Label: "setup", Mark: page.Attention, Reason: page.Partial, Cells: []page.Value{page.Text(step.Name)}, Hint: fix}
		if step.Name == "name" {
			r.Reason, r.Detail = page.Unset, page.Text("not set")
		}
		return r
	}
	if s.State != catalog.StateOK {
		return page.Row{ID: "setup", Label: "setup", Mark: pages.MarkOf(s.State), Reason: page.Late, Cells: []page.Value{page.Text("unknown")}, Hint: fix}
	}
	return row("setup", page.OK, page.Text("done"), "")
}

func environmentRows(s Overview) []page.Row {
	rows := checkRows(s.Environment.Network)
	rows = append(rows, checkRows(s.Environment.Protections)...)
	rows = append(rows, checkRows(s.Environment.Updates)...)
	if s.Tailnet.State == catalog.StatePending {
		return append(rows, pending("tailnet", catalog.TailnetProduct))
	}
	return append(rows, pageports.TailnetSummary(s.Tailnet))
}

func workspaceRows(w Workspace) []page.Row {
	rows := make([]page.Row, 0, 1+len(w.Misplaced))
	rows = append(rows, diskRow(w.Disk, w.DiskPending))
	for i, st := range w.Misplaced {
		r := page.Row{ID: st.Name + "/" + strconv.Itoa(i), Label: st.Name, Mark: page.Attention, Reason: page.Misplaced, Cells: []page.Value{page.Path(st.Path)}, Detail: page.Size(st.Bytes)}
		if st.Repair != "" {
			r.Hint = page.Cmd(st.Repair)
		}
		rows = append(rows, r)
	}
	return rows
}

func diskRow(disk []overview.Disk, waiting bool) page.Row {
	var total int64
	for _, d := range disk {
		if d.Level == catalog.LevelCache {
			total += d.Bytes
		}
	}
	if waiting {
		last := page.Pending()
		if total > 0 {
			last = page.Stale(page.Size(total))
		}
		return page.Row{ID: "disk", Label: "disk", Mark: page.Awaiting, Cells: []page.Value{last}}
	}
	if total == 0 {
		return row("disk", page.OK, page.Text("no caches"), "")
	}
	return row("disk", page.OK, page.Size(total), "reclaimable: station disk free")
}

func rowsOf(s page.Section) []page.Row {
	var out []page.Row
	for _, it := range s.Items {
		if it.Row != nil {
			out = append(out, *it.Row)
		}
	}
	return out
}

// invariant: an action says what is wrong, its value then its detail, and
// carries the row's fix; a tools or config row keeps one of the two, its
// cells being columns of their own page.
func action(section string, r page.Row) page.Row {
	act := r
	act.ID = section + "/" + r.ID
	act.Cells = nil
	var parts []page.Value
	if len(r.Cells) > 0 && !r.Cells[0].IsNone() && (section != "apps" && section != "settings" || r.Detail.IsNone()) {
		parts = append(parts, r.Cells[0])
	}
	if !r.Detail.IsNone() {
		parts = append(parts, r.Detail)
	}
	switch {
	case len(parts) == 2 && r.Reason == page.Unset && parts[0].Kind == page.KindText && parts[1].Kind == page.KindText:
		act.Detail = page.Text(parts[0].Text + " " + parts[1].Text)
	case len(parts) == 1:
		act.Detail = parts[0]
	case len(parts) > 1:
		act.Detail = page.List(parts...)
	}
	return act
}

func todo(detail []page.Section, jobs []background.Task) page.Section {
	var broken, fixes, notes, tasks []page.Row
	for _, s := range detail {
		if s.ID == "background" {
			continue
		}
		rows := rowsOf(s)
		switch s.ID {
		case "apps":
			rows = foldApps(rows)
		case "workspace":
			rows = foldMisplaced(rows)
		}
		for _, r := range rows {
			// invariant: an unregistered name is one action, the setup row's
			// `name not set → station init`; the Machine row only reads it.
			if s.ID == "machine" && r.ID == "name" {
				continue
			}
			act := action(s.ID, r)
			switch {
			case r.Mark == page.Broken:
				broken = append(broken, act)
			case r.Mark == page.Attention && act.Hint != nil:
				fixes = append(fixes, act)
			case r.Mark == page.Attention:
				notes = append(notes, act)
			}
		}
	}
	var failing []page.Row
	for _, j := range jobs {
		switch {
		case j.State == catalog.StateBroken:
			failing = append(failing, jobAction(j))
		case j.State == catalog.StateDegraded && j.Group == background.GroupStation:
			fixes = append(fixes, jobAction(j))
		case j.State == catalog.StateDegraded:
			tasks = append(tasks, jobAction(j))
		}
	}
	lists := [][]page.Row{broken, failing, fixes, notes, tasks}
	var items []page.Item
	for _, list := range lists {
		for i := range list {
			items = append(items, page.Item{Row: &list[i]})
		}
	}
	if len(items) == 0 {
		items = append(items, page.Item{Empty: &page.Empty{Noun: "actions"}})
	}
	return page.Section{ID: "todo", Title: "To do", Role: page.Actions, Fold: page.Open, Columns: []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}}, Items: items}
}

// invariant: H2 for the workspace: strays sharing a fix are one action, their
// count and their total size.
func foldMisplaced(rows []page.Row) []page.Row {
	var out, misplaced []page.Row
	for _, r := range rows {
		if r.Reason == page.Misplaced {
			misplaced = append(misplaced, r)
			continue
		}
		out = append(out, r)
	}
	if len(misplaced) < 2 {
		return append(out, misplaced...)
	}
	var total int64
	for _, r := range misplaced {
		if r.Detail.Kind == page.KindSize {
			total += r.Detail.N
		}
	}
	r := page.Row{ID: "misplaced", Label: "misplaced", Mark: page.Attention, Reason: page.Misplaced, Hint: misplaced[0].Hint,
		Detail: page.List(page.Count(int64(len(misplaced)), "misplaced folders"), page.Size(total))}
	return append(out, r)
}

func foldApps(rows []page.Row) []page.Row {
	type key struct {
		reason page.Reason
		fix    string
	}
	groups := map[key][]page.Row{}
	var order []key
	for _, r := range rows {
		if r.Mark != page.Attention {
			continue
		}
		k := key{reason: r.Reason}
		if r.Hint != nil {
			k.fix = r.Hint.Command
		}
		if _, seen := groups[k]; !seen {
			order = append(order, k)
		}
		groups[k] = append(groups[k], r)
	}
	out := make([]page.Row, 0, len(order))
	for _, k := range order {
		g := groups[k]
		if len(g) == 1 {
			out = append(out, g[0])
			continue
		}
		n := int64(len(g))
		r := page.Row{ID: string(k.reason) + "/" + k.fix, Label: "apps", Mark: page.Attention, Reason: k.reason, Hint: g[0].Hint}
		switch k.reason {
		case page.Missing:
			r.Detail = page.Count(n, "wait for "+strings.TrimPrefix(k.fix, "station apps install "))
		default:
			r.Detail = page.Count(n, string(k.reason))
		}
		out = append(out, r)
	}
	return out
}

func worst(s page.Section) page.Mark {
	for _, m := range []page.Mark{page.Broken, page.Attention, page.Awaiting} {
		for _, t := range s.Counts {
			if t.Mark == m {
				return m
			}
		}
	}
	return page.OK
}

func find(sections []page.Section, id string) page.Section {
	for _, s := range sections {
		if s.ID == id {
			return s
		}
	}
	return page.Section{}
}

func checkValue(checks []overview.Check, name string) string {
	for _, c := range checks {
		if c.Name == name && c.State == catalog.StateOK {
			return c.Value
		}
	}
	return ""
}

func networkValue(e overview.Environment) page.Value {
	parts := page.Texts(checkValue(e.Network, "local ip")).Items
	on := 0
	for _, c := range e.Protections {
		if c.State == catalog.StateOK || offByChoice(c) {
			on++
		}
	}
	if len(e.Protections) > 0 {
		parts = append(parts, page.Text(strconv.Itoa(on)+"/"+strconv.Itoa(len(e.Protections))+" on"))
	}
	return page.List(parts...)
}

func tailnetValue(t ports.Status) page.Value {
	name := t.Name
	if short, _, _ := strings.Cut(t.DNSName, "."); short != "" {
		name = short
	}
	if !t.Connected {
		state := strings.ToLower(t.BackendState)
		if name == "" && state == "" {
			state = "off"
		}
		return page.Texts(name, state)
	}
	serve := "Serve off"
	if t.ServeOn {
		serve = "Serve on"
	}
	return page.Texts(name, serve)
}

func identityValue(checks []overview.Check) page.Value {
	var parts []string
	parts = append(parts, checkValue(checks, "git-email"))
	if checkValue(checks, "git-signing") == "on" {
		parts = append(parts, "signing on")
	}
	if parts[0] == "" && len(parts) == 1 {
		return page.Text("not set")
	}
	return page.Texts(parts...)
}

func diskValue(w Workspace) page.Value {
	var cache int64
	for _, d := range w.Disk {
		if d.Level == catalog.LevelCache {
			cache += d.Bytes
		}
	}
	disk := page.Size(cache)
	disk.Unit = "reclaimable"
	if n := len(w.Misplaced); n > 0 {
		return page.List(disk, page.Count(int64(n), "misplaced"))
	}
	return disk
}

func awakeValue(s Overview) page.Value {
	if !s.Awake.Held {
		return page.Off()
	}
	words := []string{"awake"}
	if s.Awake.Lid {
		words = append(words, "lid")
	}
	return page.Texts(append(words, pages.Term(s.Awake.Until.Sub(s.GeneratedAt))+" left")...)
}

func allMachinesValue(s Overview) page.Value {
	d := s.Station.Engine
	if d.Value == "" || d.State == catalog.StateAbsent {
		if s.Machine.Name != "" {
			return page.Text(s.Machine.Name)
		}
		return page.Text(strings.ToLower(s.Machine.Hostname))
	}
	return page.Text("engine " + d.Value)
}

// invariant: one row per area, its worst mark and its value (H4), in the order
// the two columns read: Machine to Background, then Tools to Gateways.
func summary(s Overview, detail []page.Section) page.Section {
	area := func(id, label string, mark page.Mark, value page.Value) page.Item {
		return page.Item{Row: &page.Row{ID: "summary/" + id, Label: label, Mark: mark, Cells: []page.Value{value}}}
	}
	counts := func(id string) page.Value {
		sec := find(detail, id)
		parts := make([]page.Value, 0, len(sec.Counts))
		for _, t := range sec.Counts {
			parts = append(parts, page.Labeled(string(t.Mark), page.Count(int64(t.N), "")))
		}
		return page.List(parts...)
	}
	m := s.Machine
	up := ""
	if s.System.Uptime != "" {
		up = "up " + s.System.Uptime
	}
	machine := page.Texts(machineWord(m.Kind), m.OS, up)
	items := []page.Item{
		area("machine", "Machine", worst(find(detail, "machine")), machine),
		area("network", "Network", worst(find(detail, "environment")), networkValue(s.Environment)),
		area("tailnet", catalog.TailnetProduct, pages.MarkOf(s.Tailnet.State), tailnetValue(s.Tailnet)),
		area("identity", "Identity", worst(find(detail, "identity")), identityValue(s.Identity)),
	}
	if len(s.Background) > 0 {
		items = append(items, area("background", "Background", backgroundMark(s.Background), backgroundValue(s.Background)))
	}
	right := len(items)
	items = append(items,
		area("apps", "Apps", worst(find(detail, "apps")), counts("apps")),
		area("settings", "Settings", worst(find(detail, "settings")), counts("settings")),
		area("disk", "Disk", page.OK, diskValue(s.Workspace)),
		area("all-machines", "All machines", pages.MarkOf(s.Station.Engine.State), allMachinesValue(s)),
	)
	if s.AwakeKnown {
		items = append(items, area("awake", "Awake", page.OK, awakeValue(s)))
	}
	if len(s.Gateways) > 0 {
		items = append(items, area("gateways", "Gateways", worst(find(detail, "gateways")), counts("gateways")))
	}
	return page.Section{ID: "summary", Title: "Summary", Role: page.Summary, Fold: page.Open, Uncounted: true, Break: right, Columns: []page.Column{{Key: "area", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}, Items: items}
}

func machineWord(k catalog.MachineKind) string {
	switch k {
	case catalog.Mac:
		return "Mac"
	case catalog.Linux:
		return "Linux VM"
	case catalog.MacOS:
		return "macOS VM"
	}
	return string(k)
}
