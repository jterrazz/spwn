package ports

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
	"github.com/jterrazz/studio-station/src/ports"
)

const KeyStopSharing = "x"

const (
	ClassNeedsYou = "needs you"
	ClassPublic   = "public"
	ClassPrivate  = "private"
	ClassLocal    = "this machine only"
)

func Columns() []page.Column {
	return []page.Column{
		{Key: "vm", Kind: page.KindText, Keep: 100},
		{Key: "port", Label: "port", Kind: page.KindPortMap, Keep: 80},
		{Key: "class", Label: "class", Kind: page.KindText, Keep: 85},
		{Key: "url", Label: "url", Kind: page.KindURL, Keep: 40},
		{Key: "state", Label: "state", Kind: page.KindText, Keep: 70},
		{Key: "process", Label: "process", Kind: page.KindText, Keep: 20, Whole: true},
	}
}

func List(l ports.Listing, cat catalog.Catalog, thisMac page.Context) page.Page {
	return Listed(l, cat, thisMac, false)
}

func Listed(l ports.Listing, cat catalog.Catalog, thisMac page.Context, all bool) page.Page {
	// invariant: OU-01: what needs you comes first, before the Tailscale
	// line, as on every page.
	sections := Sections(l.AllMachines, cat, thisMac, all)
	if line := TailnetLine(l.Tailnet); line != nil {
		at := 0
		if len(sections) > 0 && sections[0].ID == "needs-you" {
			at = 1
		}
		sections = slices.Insert(sections, at, *line)
	}
	sections = append([]page.Section{firstLine(sections, whose(l.AllMachines, thisMac))}, sections...)
	return page.Seal(page.Page{ID: "ports", Command: "ports", Context: thisMac, Sections: sections, Actions: Actions(l.AllMachines)})
}

func TailnetLine(s ports.Status) *page.Section {
	// value: the tailnet's line a list of ports begins with: whether the
	// tailnet carries this Mac's shares; nil inside a VM, whose Mac says it.
	if s.State == "" {
		return nil
	}
	row := page.Row{ID: "tailnet", Label: catalog.TailnetProduct}
	switch {
	case s.State == catalog.StatePending:
		row.Mark = page.Awaiting
	case s.State == catalog.StateAbsent:
		row.Mark, row.Reason, row.Cells = page.Absent, page.Missing, []page.Value{page.Text("not installed")}
		row.Hint = repair(s)
	case !s.Connected:
		row.Mark, row.Reason = page.Broken, page.Down
		row.Cells = []page.Value{page.Text("is signed out on This Mac: ports are local only")}
		row.Hint = repair(s)
	case !s.ServeOn:
		row.Mark, row.Reason = page.Attention, page.Degraded
		row.Cells = []page.Value{page.Texts("signed in", s.DNSName, "ports are local only")}
		row.Detail = page.Text(catalog.ServeOffDetail)
	default:
		row.Mark, row.Cells = page.OK, []page.Value{page.Texts("signed in", s.DNSName, "ports are shared")}
	}
	return &page.Section{ID: "tailnet", Title: catalog.TailnetProduct, Role: page.Banner,
		Columns: []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "state", Kind: page.KindText, Keep: 90}},
		Items:   []page.Item{{Row: &row}}}
}

func Sections(f ports.AllMachines, cat catalog.Catalog, thisMac page.Context, all bool) []page.Section {
	// invariant: SPEC 5.6: what needs you first, then Public, then Private;
	// what listens on its machine only is a count, unfolded by all; each Mac's
	// reserved ports one line.
	var red, public, private, local, notes []page.Item
	for _, h := range f.Macs {
		private = append(private, macRows(h)...)
		if !h.Reachable {
			continue
		}
		shared := map[string]bool{}
		for _, s := range h.Shared {
			row := Row(s)
			shared[row.ID] = true
			private = append(private, page.Item{Row: &row})
		}
		for _, p := range h.Ports {
			switch {
			case p.Red():
				row := named(RedRow(p), thisMac)
				red = append(red, page.Item{Row: &row})
			case p.Declared != nil:
				row := named(Answering(PublicRow(p, cat), p), thisMac)
				public = append(public, page.Item{Row: &row})
			case shared[p.Key()]:
			case p.Reach == ports.ThisMachine:
				row := named(Answering(LocalRow(p), p), thisMac)
				local = append(local, page.Item{Row: &row})
			default:
				row := named(Answering(PrivateRow(p), p), thisMac)
				private = append(private, page.Item{Row: &row})
			}
		}
		if h.Range != nil {
			notes = append(notes, page.Item{Note: &page.Note{Text: reserved(h, thisMac)}})
		}
	}
	var out []page.Section
	if len(red) > 0 {
		out = append(out, page.Section{ID: "needs-you", Title: "Needs you: " + catalog.ReachableOutside + ", not declared public", PipeCounts: true, Columns: exposedColumns(), Items: red})
	}
	if len(public) > 0 {
		out = append(out, page.Section{ID: "public", Title: "Public: reachable from anywhere else, internet or LAN, declared", PipeCounts: true, Columns: exposedColumns(), Items: public})
	}
	if len(private) == 0 {
		private = append(private, page.Item{Empty: &page.Empty{Noun: "shared ports", Hint: page.Cmd("station machines create")}})
	}
	out = append(out, page.Section{ID: "private", Title: "Private: shared with you on " + catalog.TailnetProduct, PipeCounts: true, Columns: Columns(), Items: private})
	if all && len(local) > 0 {
		out = append(out, page.Section{ID: "local-only", Title: "Listens on its machine only", PipeCounts: true, Columns: exposedColumns(), Items: local})
	}
	if !all && len(local) > 0 {
		notes = append(notes, page.Item{Note: &page.Note{Text: localOnly(len(local))}})
	}
	if len(notes) > 0 {
		out = append(out, page.Section{ID: "ranges", Title: "Reserved ports", Role: page.Banner, Uncounted: true, Columns: Columns(), Items: notes})
	}
	return out
}

func localOnly(n int) string {
	if n == 1 {
		return "1 more listens on its machine only"
	}
	return strconv.Itoa(n) + " more listen on their machine only"
}

// value: a host's own rows: it does not answer, or its engine does not run.
func macRows(h ports.MacPorts) []page.Item {
	var row page.Row
	switch {
	case !h.Reachable:
		row = page.Row{ID: h.Name, Label: h.Name, Mark: page.Attention, Reason: page.Unreachable,
			Cells: []page.Value{page.None(), page.None(), page.None(), page.Text("does not answer")},
			Hint:  page.Cmd("station machines")}
	case h.ThisMac && !h.Engine.Running:
		reason, word, fix := page.Stopped, "not shared while the engine is stopped", "station engine start"
		if !h.Engine.Installed {
			reason, word, fix = page.Missing, "the engine is not installed", "station engine start"
		}
		row = page.Row{ID: "engine", Label: "engine", Mark: page.Attention, Reason: reason,
			Cells: []page.Value{page.None(), page.None(), page.None(), page.Text(word)},
			Hint:  page.Cmd(fix)}
	default:
		return nil
	}
	return []page.Item{{Row: &row}}
}

func exposedColumns() []page.Column {
	return []page.Column{
		{Key: "machine", Kind: page.KindText, Keep: 100},
		{Key: "port", Label: "port", Kind: page.KindPort, Keep: 90},
		{Key: "class", Label: "class", Kind: page.KindText, Keep: 85},
		{Key: "what", Label: "what", Kind: page.KindText, Keep: 60},
		{Key: "listens", Label: "listens on", Kind: page.KindText, Keep: 50},
		{Key: "declared", Label: "declared", Kind: page.KindText, Keep: 70},
	}
}

func ListensOn(p ports.Port) page.Value {
	// value: the addresses a port listens on, in the owner's words: every
	// interface, the LAN, the tailnet, this machine.
	words := make([]string, 0, len(p.Binds))
	for _, b := range p.Binds {
		port := strconv.Itoa(p.Port)
		switch addr := strings.Trim(b, "[]"); {
		case addr == "" || addr == "*" || addr == "0.0.0.0" || addr == "::":
			words = append(words, "*:"+port+" all interfaces")
		case ports.ReachOf(b, ports.OnMac) == ports.ThisMachine:
			words = append(words, b+":"+port+" this machine")
		case ports.ReachOf(b, ports.OnMac) == ports.Tailnet:
			words = append(words, b+":"+port+" "+catalog.TailnetProduct)
		default:
			words = append(words, b+":"+port+" LAN")
		}
	}
	return page.Texts(words...)
}

// value: a port's row names its machine as the interface does on a
// terminal (SPEC §2.5: This Mac, mac-mini); its records keep the full name.
func named(r page.Row, thisMac page.Context) page.Row {
	if short := page.MacName(r.Label, thisMac.Machine); short != r.Label {
		r.Short = short
	}
	return r
}

func RedRow(p ports.Port) page.Row {
	row := page.Row{ID: p.Key(), Label: p.Machine.Name, Mark: page.Broken, Reason: page.Exposed,
		Cells: []page.Value{page.Port(int64(p.Port)), page.Text(ClassNeedsYou), page.Text(p.What()), ListensOn(p), page.Text("declared no")}}
	if k, ok := catalog.KnownPortOf(p.Port, p.Process); ok && k.Off != "" {
		row.Hint = page.Cmd(k.Off)
	}
	return row
}

func PublicRow(p ports.Port, cat catalog.Catalog) page.Row {
	// invariant: SPEC 5.6: a declared port is OK when its route is seen,
	// attention with what is missing when it is not.
	row := page.Row{ID: p.Key(), Label: p.Machine.Name, Mark: page.OK,
		Cells: []page.Value{page.Port(int64(p.Port)), page.Text(ClassPublic), page.Text(p.What()), ListensOn(p), page.Texts("declared", p.Declared.Via)}}
	v := p.Verified
	switch {
	case v == nil:
	case !v.Seen:
		row.Mark, row.Reason = page.Attention, page.Degraded
		row.Cells[4] = page.Text("declared, but " + v.Missing)
	case p.Declared.By != "":
		r, _ := catalog.RouteNamed(p.Declared.Via)
		row.Cells[4] = page.Texts(r.Label, cat.DeclaredBy(p.Declared.By), "by Settings")
	default:
		r, _ := catalog.RouteNamed(p.Declared.Via)
		row.Cells[4] = page.Texts(r.Label, r.Seen)
	}
	return row
}

func Answering(r page.Row, p ports.Port) page.Row {
	// invariant: SPEC B5.17: a port follows its VM's state: greyed
	// unreachable while it is down, starting while it boots again.
	switch p.Answer {
	case ports.Unreachable:
		r.Mark, r.Reason, r.Detail = page.Attention, page.Unreachable, page.Text(catalog.UnreachableDetail(p.Machine.Name))
	case ports.Starting:
		r.Mark, r.Reason, r.Detail = page.Attention, page.Waiting, page.Text(catalog.StartingDetail(p.Machine.Name))
	case ports.Answers:
	}
	return r
}

func LocalRow(p ports.Port) page.Row {
	return page.Row{ID: p.Key(), Label: p.Machine.Name, Mark: page.OK,
		Cells: []page.Value{page.Port(int64(p.Port)), page.Text(ClassLocal), page.Text(p.What()), ListensOn(p), page.None()}}
}

func PrivateRow(p ports.Port) page.Row {
	// value: a port reachable through the tailnet only that station does not
	// share: a VM's own tailnet device, its Mac, or the Mac's tailnet address.
	how := "on " + catalog.TailnetProduct
	switch {
	case p.Device:
		how = "own " + catalog.TailnetProduct + " device: " + p.Machine.Name
	case p.Machine.On != "":
		how = "behind " + p.Machine.On + ", not shared"
	}
	return page.Row{ID: p.Key(), Label: p.Machine.Name, Mark: page.OK,
		Cells: []page.Value{page.Port(int64(p.Port)), page.Text(ClassPrivate), page.None(), page.Text(how), page.Text(p.What())}}
}

func reserved(h ports.MacPorts, thisMac page.Context) string {
	r, mac := h.Range, page.MacName(h.Name, thisMac.Machine)
	if mac == "" && h.ThisMac {
		mac = "This Mac"
	}
	size := r.High - r.Low + 1
	return "Reserved on " + mac + ": " + strconv.Itoa(r.Reserved) + " of " + strconv.Itoa(size) +
		" (" + strconv.Itoa(r.Low) + "-" + strconv.Itoa(r.High) + "), " + strconv.Itoa(r.Free) + " free"
}

func Actions(f ports.AllMachines) []page.Action {
	// invariant: stop sharing binds only a port's row, never a Mac's or the engine's.
	var shared []string
	for _, h := range f.Macs {
		if !h.Reachable {
			continue
		}
		for _, s := range h.Shared {
			shared = append(shared, Row(s).ID)
		}
	}
	if len(shared) == 0 {
		return nil
	}
	return []page.Action{{Key: KeyStopSharing, Label: "stop sharing", On: page.OnRow, Rows: shared, Priority: 1, Mutates: true, Confirm: &page.Confirm{Yes: "stop sharing", Danger: true}}}
}

// value: a port's state in the owner's words (SPEC 5.6), the mark it reads
// with, and the fix it waits for.
func stateOf(s ports.SharedPort) (page.Mark, page.Reason, page.Value, *page.Hint) {
	reserved := "reserved " + strconv.Itoa(s.Reserved)
	switch {
	case s.State == ports.PhaseNotAllowed:
		return page.Attention, page.Waiting, page.Text(s.Detail), page.Cmd("station ports share " + s.VM + " " + strconv.Itoa(s.GuestPort))
	case s.State == ports.PhaseRefused || s.Reserved == 0:
		return page.Attention, page.Waiting, page.Texts("waiting", s.Detail), page.Cmd("station ports stop-sharing " + s.VM + " <port>")
	case s.State == ports.PhaseUnknown:
		return page.Attention, page.Late, page.Texts("not shared while the engine is stopped", reserved), nil
	case s.State == ports.PhaseWaiting && strings.HasPrefix(s.Detail, catalog.StoppedDetail(s.VM)):
		return page.Absent, page.Stopped, page.Texts(s.Detail, reserved), nil
	case s.State == ports.PhaseWaiting:
		return page.Absent, page.Waiting, page.Texts("not listening", reserved), nil
	case s.Detail == catalog.IssuingDetail:
		return page.Attention, page.Waiting, page.Text(s.Detail), nil
	case s.Detail != "":
		return page.Attention, page.Degraded, page.Texts("local only", s.Detail), nil
	}
	return page.OK, "", page.Text("shared"), nil
}

func rowID(vm string, port int) string { return vm + ":" + strconv.Itoa(port) }

func Row(s ports.SharedPort) page.Row {
	url := page.URL(s.URLs.Public)
	if s.Reserved == 0 || s.URLs.Public == "" {
		url = page.None()
	}
	port := page.Port(int64(s.GuestPort))
	if s.Reserved != 0 {
		port = page.PortMap(int64(s.GuestPort), int64(s.Reserved))
	}
	mark, reason, state, hint := stateOf(s)
	return page.Row{ID: rowID(s.VM, s.GuestPort), Label: s.VM, Mark: mark, Reason: reason,
		Cells:  []page.Value{port, page.Text(classOf(s)), url, state, page.Text(s.Process)},
		Detail: page.Text(s.Profile), Hint: hint}
}

// value: who a VM's shared port reaches, in the class words: the
// internet's for one open to public, its Mac's alone for one open to
// local, else the owner's devices.
func classOf(s ports.SharedPort) string {
	switch catalog.Audience(s.To) {
	case catalog.AudiencePublic:
		return ClassPublic
	case catalog.AudienceLocal:
		return ClassLocal
	case catalog.AudiencePrivate:
	}
	return ClassPrivate
}

func Link(s ports.SharedPort) page.Link {
	l := page.Link{Label: strconv.Itoa(s.GuestPort), Lead: page.Port(int64(s.GuestPort)), Value: page.URL(s.URLs.Public), Detail: page.Text(s.Process), Page: "ports", Row: rowID(s.VM, s.GuestPort)}
	if s.Reserved != 0 {
		l.Lead = page.PortMap(int64(s.GuestPort), int64(s.Reserved))
	}
	return l
}

func Shared(s ports.SharedPort, thisMac page.Context) page.Page {
	row := pages.Done(rowID(s.VM, s.GuestPort), catalog.StateOK, "")
	row.Cells = []page.Value{page.URL(s.URLs.Public)}
	return pages.Outcome("ports share "+strconv.Itoa(s.GuestPort), thisMac,
		[]page.Column{{Key: "port", Kind: page.KindText, Keep: 100}, {Key: "url", Kind: page.KindURL, Keep: 90}}, []page.Row{row}, "shared ports")
}

func Declared(d ports.Declaration, thisMac page.Context) page.Page {
	r, _ := catalog.RouteNamed(d.Via)
	row := pages.Done(d.Key(), catalog.StateOK, "declared public ("+r.Label+")")
	row.Label = strconv.Itoa(d.Port) + " on " + d.Machine
	return pages.Outcome("ports declare", thisMac, nil, []page.Row{row}, "declared ports")
}

func Undeclared(d ports.Declaration, thisMac page.Context) page.Page {
	row := pages.Done(d.Key(), catalog.StateOK, "not declared")
	row.Label = strconv.Itoa(d.Port) + " on " + d.Machine
	return pages.Outcome("ports undeclare", thisMac, nil, []page.Row{row}, "declared ports")
}

func Stopped(r ports.Reservation, thisMac page.Context) page.Page {
	return pages.Outcome("ports stop-sharing", thisMac, nil, []page.Row{pages.Done(r.Key(), catalog.StateOK, "not shared")}, "shared ports")
}

// value: the first line: whose ports, how many, and how many need you
// (vision 07 "station ports": `mac-mini and infra · 9 ports · 4 need you`).
// value: whose ports a list holds: every Mac's, another Mac's, This Mac's.
func whose(f ports.AllMachines, thisMac page.Context) string {
	switch {
	case len(f.Macs) > 1:
		return "Every Mac"
	case len(f.Macs) == 1 && thisMac.Machine != "" && f.Macs[0].Name != thisMac.Machine:
		return page.MacName(f.Macs[0].Name, thisMac.Machine)
	case thisMac.Machine != "" && thisMac.Kind != string(machine.Computer):
		return thisMac.Machine
	}
	return "This Mac"
}

func firstLine(sections []page.Section, label string) page.Section {
	n, needs := 0, 0
	for _, s := range sections {
		if s.ID == "tailnet" || s.ID == "ranges" {
			continue
		}
		for _, it := range s.Items {
			if it.Row == nil && it.Link == nil {
				continue
			}
			n++
			if it.Row != nil && (it.Row.Mark == page.Broken || it.Row.Mark == page.Attention) {
				needs++
			}
		}
	}
	words := []string{strconv.Itoa(n) + " ports"}
	if n == 1 {
		words[0] = "1 port"
	}
	switch needs {
	case 0:
		words = append(words, "nothing needs you")
	case 1:
		words = append(words, "1 needs you")
	default:
		words = append(words, strconv.Itoa(needs)+" need you")
	}
	row := page.Row{ID: "summary/ports", Label: label, Mark: page.OK, Cells: []page.Value{page.Texts(words...)}}
	return page.Section{ID: "summary", Title: "Summary", Role: page.Summary, Fold: page.Open, Uncounted: true, Columns: []page.Column{{Key: "area", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}, Items: []page.Item{{Row: &row}}}
}

func Links(mode string, thisMac page.Context) page.Page {
	// value: the outcome of HTTPS links turned on or off: the link the
	// shares of this Mac now get.
	detail := "shared ports get an https:// link"
	if !ports.HTTPSLinks(mode) {
		detail = "shared ports get an http:// link"
	}
	row := pages.Done("https", catalog.StateOK, detail)
	row.Label = "HTTPS links " + map[bool]string{true: "on", false: "off"}[ports.HTTPSLinks(mode)]
	return pages.Outcome("ports https", thisMac, nil, []page.Row{row}, "HTTPS links")
}
