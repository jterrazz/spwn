package accounts

import (
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
)

const (
	KeyShare    = "s"
	KeyTakeBack = "t"
	KeyReload   = "R"
	EngineRow   = "engine"
)

func columns() []page.Column {
	return []page.Column{
		{Key: "provider", Kind: page.KindText, Keep: 100},
		{Key: "identity", Kind: page.KindIdentity, Keep: 60},
		{Key: "origin", Kind: page.KindText, Keep: 90},
		{Key: "scope", Kind: page.KindList, Keep: 70, Whole: true},
		{Key: "detail", Kind: page.KindText, Keep: 50},
	}
}

func label(provider string) string {
	for _, p := range catalog.Providers {
		if p.ID == provider {
			return p.Label
		}
	}
	return provider
}

func mark(a accounts.Account) page.Mark { return pages.MarkOf(a.State) }

func Reason(r string) page.Reason {
	switch r {
	case accounts.ReasonUnset:
		return page.Unset
	case accounts.ReasonMissing:
		return page.Missing
	case accounts.ReasonRefused:
		return page.Refused
	case accounts.ReasonLate:
		return page.Late
	case accounts.ReasonFailing:
		return page.Failing
	case accounts.ReasonExpiring:
		return page.Expiring
	case accounts.ReasonUnreachable:
		return page.Unreachable
	}
	return ""
}

func detail(a accounts.Account, more ...string) page.Value {
	return page.Texts(append(append(more, a.Source, a.Detail), a.Notes...)...)
}

// invariant: a copy shared with a VM says so (SPEC §2.2: `lend` retired);
// the contract keeps its origin values.
func origin(a accounts.Account) page.Value {
	switch a.Origin {
	case accounts.OriginBoth:
		return page.Texts(accounts.OriginOwn, shared)
	case accounts.OriginShared:
		return page.Text(shared)
	}
	return page.Text(a.Origin)
}

const shared = "shared"

func identity(a accounts.Account) page.Value {
	if len(a.Identity) == 0 {
		return page.None()
	}
	ids := make([]page.Value, 0, len(a.Identity))
	for _, id := range a.Identity {
		ids = append(ids, page.Identity(id))
	}
	if len(ids) == 1 {
		return ids[0]
	}
	return page.List(ids...)
}

func until(a accounts.Account, now time.Time) page.Value {
	if a.Shared == nil {
		return page.None()
	}
	return page.Until(a.Shared.Until, now)
}

func row(m accounts.Machine, a accounts.Account, now time.Time) page.Row {
	r := page.Row{ID: accounts.RowID(m.Name, a.Provider), Label: a.Provider, Short: label(a.Provider), Mark: mark(a)}
	if a.State == catalog.StatePending {
		r.Cells = []page.Value{page.Pending()}
		return r
	}
	// value: the eye reads a machine's first identity in the column and the
	// rest beside its detail (git's e-mail); the records keep them together.
	id, last := identity(a), detail(a)
	if len(a.Identity) > 1 {
		id, last = page.Identity(a.Identity[0]), detail(a, a.Identity[1:]...)
	}
	if a.Shared != nil {
		last = until(a, now)
	}
	scope := page.Texts(a.Scopes...)
	if a.Provider == catalog.ProviderSSH {
		// value: ssh's scopes are its keys' comments, as many as the agent
		// holds; the list reads them as the identity's count and keeps them
		// for the row's detail, so they never take the room GitHub's scopes
		// and Claude's plan need (§13.9).
		scope, r.Detail = page.None(), scope
	}
	r.Cells = []page.Value{id, origin(a), scope, last}
	if r.Mark == page.Attention || r.Mark == page.Broken || r.Mark == page.Absent {
		r.Reason = Reason(a.Reason)
	}
	if a.Hint != "" {
		r.Hint = page.Cmd(a.Hint)
	}
	return r
}

func worst(rows []page.Row) page.Mark {
	m := page.OK
	for _, r := range rows {
		switch {
		case r.Mark == page.Awaiting:
			return page.Awaiting
		case r.Mark == page.Broken:
			m = page.Broken
		case r.Mark == page.Attention && m != page.Broken:
			m = page.Attention
		}
	}
	return m
}

func head(m accounts.Machine, rows []page.Row) page.Row {
	h := page.Row{ID: m.Name, Label: m.Name, Mark: worst(rows)}
	kind := m.Kind
	switch m.Kind {
	case accounts.KindVM:
		kind = m.Lifetime
	case accounts.KindComputer:
		kind = "Mac"
	}
	cells := []page.Value{page.Text(kind)}
	switch {
	case m.Pending:
		h.Mark = page.Awaiting
	case m.Kind == accounts.KindVM && !m.Running:
		h.Detail = page.Text("stopped")
	case m.Guest == accounts.GuestAbsent:
		h.Mark, h.Reason, h.Detail = page.Attention, page.Unreachable, page.Text("guest absent: rights from the record only")
	case m.Guest != "":
		h.Detail = page.Text("guest " + m.Guest)
	}
	if h.Mark == page.Attention && h.Reason == "" || h.Mark == page.Broken {
		h.Reason = page.Degraded
	}
	h.Cells = cells
	return h
}

// value: a VM lists the rows that say something (a right read, a lend, a
// fault); the providers merely absent there fold away under one heading.
func shown(m accounts.Machine, a accounts.Account) bool {
	return m.Kind != accounts.KindVM || a.State != catalog.StateAbsent || a.Shared != nil
}

func group(m accounts.Machine, now time.Time) page.Item {
	var rows, folded []page.Row
	for _, a := range m.Accounts {
		r := row(m, a, now)
		if shown(m, a) {
			rows = append(rows, r)
		} else {
			folded = append(folded, r)
		}
	}
	g := &page.Group{Head: head(m, rows), Fold: page.Open}
	for i := range rows {
		g.Items = append(g.Items, page.Item{Row: &rows[i]})
	}
	if len(folded) > 0 {
		sub := &page.Group{Head: page.Row{ID: m.Name + "/absent", Label: "not set here"}, Fold: page.Folded}
		for i := range folded {
			sub.Items = append(sub.Items, page.Item{Row: &folded[i]})
		}
		g.Items = append(g.Items, page.Item{Group: sub})
	}
	if len(g.Items) == 0 {
		g.Items = append(g.Items, page.Item{Empty: &page.Empty{Noun: "accounts"}})
	}
	return page.Item{Group: g}
}

func engineRow(state string) page.Row {
	return page.Row{ID: EngineRow, Label: EngineRow, Mark: page.Attention, Reason: page.Stopped, Cells: []page.Value{page.Text(state)}, Hint: page.Cmd("station engine start")}
}

func Section(r accounts.Report) page.Section {
	r = r.Resolved()
	s := page.Section{ID: "accounts", Title: "Accounts", Columns: columns()}
	vms := 0
	for _, m := range r.Machines {
		if m.Kind == accounts.KindVM && !m.ThisMac {
			vms++
			if stopped(r) {
				continue
			}
		}
		s.Items = append(s.Items, group(m, r.Now))
	}
	switch {
	case stopped(r):
		d := engineRow(r.Engine)
		s.Items = append(s.Items, page.Item{Row: &d})
	case vms == 0 && len(r.Machines) > 0 && r.Machines[0].Kind == accounts.KindComputer:
		s.Items = append(s.Items, page.Item{Note: &page.Note{Text: "no VMs: accounts shared with a VM appear here"}})
	}
	return s
}

func stopped(r accounts.Report) bool {
	return r.Engine == accounts.EngineStopped || r.Engine == accounts.EngineAbsent
}

func Page(r accounts.Report, thisMac page.Context) page.Page {
	p := page.Page{ID: "accounts", Command: "accounts", Context: thisMac, Sections: []page.Section{firstLine(r, thisMac), Section(r)}}
	if len(r.Machines) == 1 && r.Machines[0].Kind == accounts.KindComputer && !stopped(r) {
		p.Hints = []page.Hint{*page.Cmd("station machines create")}
	}
	p.Actions = Actions()
	return page.Seal(p)
}

func Actions() []page.Action {
	return []page.Action{
		{Key: KeyShare, Label: "share with a VM", On: page.OnRow, Priority: 1, Mutates: true},
		{Key: KeyTakeBack, Label: "take back", On: page.OnRow, Priority: 2, Mutates: true, Confirm: &page.Confirm{Yes: "take back"}},
		{Key: KeyReload, Label: "reload", On: page.OnPage, Priority: 3},
	}
}

// value: the first line: the machine, its accounts, what needs you and the
// VMs its accounts are shared with.
func firstLine(r accounts.Report, thisMac page.Context) page.Section {
	r = r.Resolved()
	label := "This Mac"
	if thisMac.Machine != "" && thisMac.Kind != string(accounts.KindComputer) {
		label = thisMac.Machine
	}
	var words []string
	shared := 0
	for _, m := range r.Machines {
		if !m.ThisMac {
			for _, a := range m.Accounts {
				if a.Shared != nil {
					shared++
					break
				}
			}
			continue
		}
		in, needs, reading := 0, 0, false
		for _, a := range m.Accounts {
			switch mark(a) {
			case page.OK:
				in++
			case page.Broken, page.Attention:
				needs++
			case page.Awaiting:
				reading = true
			case page.Unjudged, page.Absent, page.OffHere, page.NotForThisMachine, page.NotKnown:
			}
		}
		switch len(m.Accounts) {
		case 0:
			words = append(words, "no account yet")
		case 1:
			words = append(words, "1 account", strconv.Itoa(in)+" signed in")
		default:
			words = append(words, strconv.Itoa(len(m.Accounts))+" accounts", strconv.Itoa(in)+" signed in")
		}
		switch {
		case reading:
			words = append(words, "reading")
		case needs == 1:
			words = append(words, "1 needs you")
		case needs > 1:
			words = append(words, strconv.Itoa(needs)+" need you")
		}
	}
	switch {
	case shared == 1:
		words = append(words, "shared with 1 VM")
	case shared > 1:
		words = append(words, "shared with "+strconv.Itoa(shared)+" VMs")
	}
	row := page.Row{ID: "summary/accounts", Label: label, Mark: page.OK, Cells: []page.Value{page.Texts(words...)}}
	return page.Section{ID: "summary", Title: "Summary", Role: page.Summary, Fold: page.Open, Uncounted: true, Columns: []page.Column{{Key: "area", Kind: page.KindText, Keep: 100}, {Key: "value", Kind: page.KindList, Keep: 60}}, Items: []page.Item{{Row: &row}}}
}
