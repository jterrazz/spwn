package menubar

import (
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

const (
	AccountsID page.ID = "menubar.accounts"

	SectionNeedsYou = "needs-you"
	SectionOn       = "on:"
	SectionSilent   = "silent"
	SectionNone     = "none"

	RowGrant = "grant-password"

	ActionGrant        = "grant"
	ActionSignIn       = "sign-in:"
	ActionRemoveGrant  = "remove-grant:"
	VerbWaiting        = "Waiting"
	VerbRemove         = "Remove"
	GrantLabel         = "Grant a password"
	GrantStays         = "until you remove it"
	FullListInTerminal = "Full list in Terminal"

	CallGrant       = "grants.add"
	CallRemoveGrant = "grants.remove"
)

type AccountsOf struct {
	// value: the Mac whose accounts the page lists, and with VM that VM's
	// alone; both empty is every Mac.
	Mac, VM string
	// value: the level each gesture asks at, the calls' own rule.
	Asks Levels
}

func Accounts(s SignIns, of AccountsOf, thisMac page.Context) page.Page {
	// invariant: Accounts · All machines: its header and facts (the
	// sign-ins, the grants); what needs you first, each sign-in a machine
	// is out of with its verb; then each machine's sign-ins and grants, a
	// grant with Remove; the footer lists them all in Terminal and grants
	// a password of This Mac's.
	if of.Mac != "" {
		s = s.within(of.Mac, of.VM)
	}
	b := accountsBody{s: s, self: thisMac.Machine, of: of}
	where := "All machines"
	if of.Mac != "" {
		where = b.name(of.Mac)
	}
	if of.VM != "" {
		where = of.VM
	}
	p := page.Page{ID: AccountsID, Command: "accounts", Context: thisMac, Header: &page.Header{Back: "All machines", Glyph: symbol("key", "")}}
	p.Sections = append([]page.Section{{ID: SectionSummary, Title: "Accounts", Role: page.Summary, Caption: b.facts(where)}}, b.sections()...)
	p.Actions = b.actions()
	full := page.Action{ID: ActionFullList, Label: FullListInTerminal, Icon: "arrow.up.forward", On: page.OnPage, Priority: 2, Run: &page.Run{Terminal: true, Argv: []string{"accounts", "--all-machines"}}}
	if of.Mac != "" {
		full.Run.Argv = onMac([]string{"accounts"}, of.Mac, thisMac.Machine)
	}
	p.Actions = append(p.Actions, full)
	p.Footer = &page.Footer{Kind: page.FooterTerminals, Buttons: []string{ActionFullList}}
	if sheets := b.grantSheets(); of.VM == "" && len(sheets) > 0 {
		sheets[0].On, sheets[0].Rows, sheets[0].Icon = page.OnPage, nil, "plus"
		p.Actions = append(p.Actions, sheets[0])
		if of.Mac == "" {
			p.Actions = append(p.Actions, sheets[1:]...)
		}
		p.Footer.Buttons = append(p.Footer.Buttons, sheets[0].ID)
	}
	for _, m := range b.s.Machines {
		for _, r := range m.Rows {
			if remove, ok := b.remove(m, r); ok {
				withRowMenu(&p, rowOf(m.Name, r.Program), grantMenu(m.Name, r, remove))
			}
		}
	}
	return page.Seal(p)
}

type accountsBody struct {
	s    SignIns
	self string
	of   AccountsOf
	// value: the segment of a machine page the sections belong to; empty
	// on the Accounts page.
	segment string
}

func (b accountsBody) name(mac string) string { return page.MacName(mac, b.self) }

func (b accountsBody) id(s string) string {
	if b.segment == "" {
		return s
	}
	return b.segment + ":" + s
}

func rowOf(name string, p accounts.Program) string { return name + "/" + string(p) }

// value: the page's facts: where, the sign-ins, the grants.
func (b accountsBody) facts(where string) []page.Value {
	return []page.Value{page.Text(where), page.Text(counted(b.s.signedIn(), "sign-in", "sign-ins")), page.Text(counted(b.s.grants(), "grant", "grants"))}
}

func (b accountsBody) sections() []page.Section {
	var out []page.Section
	if needs, _ := b.needs(); len(needs) > 0 {
		count := page.Text(strconv.Itoa(len(needs)))
		count.Signal = page.SignalWaits
		out = append(out, page.Section{ID: b.id(SectionNeedsYou), Title: "Needs you", Segment: b.segment, Card: page.CardFlush, Uncounted: true, Columns: columns(), Caption: []page.Value{count}, Items: needs})
	}
	var silent []page.Item
	for _, m := range b.s.Machines {
		if m.Silent != "" {
			row := page.Row{ID: m.Name, Label: m.Name, Short: b.label(m), Mark: page.NotKnown, Reason: page.Unreachable, Cells: []page.Value{page.Text(m.Silent)}}
			silent = append(silent, page.Item{Row: &row})
			continue
		}
		if s, ok := b.on(m); ok {
			out = append(out, s)
		}
	}
	if len(silent) > 0 {
		out = append(out, page.Section{ID: b.id(SectionSilent), Title: "Not read", Segment: b.segment, Uncounted: true, Columns: columns(), Items: silent})
	}
	if len(out) == 0 {
		out = []page.Section{{ID: b.id(SectionNone), Title: "Accounts", Segment: b.segment, Columns: columns(), Items: []page.Item{{Empty: &page.Empty{Noun: "accounts"}}}}}
	}
	return out
}

func (b accountsBody) label(m MachineSignIns) string {
	if m.VM {
		return m.Name
	}
	return b.name(m.Name)
}

// value: one machine's card: each program signed in there, by its own
// sign-in or granted, with what it says; on a VM of This Mac, Grant a
// password under them, and the line that says none is granted when none
// is.
func (b accountsBody) on(m MachineSignIns) (page.Section, bool) {
	s := page.Section{ID: b.id(SectionOn + m.Name), Title: "On " + b.label(m), Segment: b.segment, Card: page.CardFlush, Columns: columns()}
	for _, r := range m.Rows {
		if r.State != accounts.ProgramSignedIn {
			continue
		}
		row := page.Row{ID: rowOf(m.Name, r.Program), Label: r.Program.Label(), Mark: page.OK, Glyph: logo(r.Program)}
		if words := signInWords(r); len(words) > 0 {
			row.Detail = page.Texts(words...)
		}
		if r.How == accounts.HowGranted {
			row.Mac, row.Cells = m.Name, []page.Value{page.Text("granted")}
			if remove, ok := b.remove(m, r); ok {
				row.Menu = []string{remove.ID}
			}
		}
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	s.Caption = []page.Value{page.Text(strconv.Itoa(len(s.Items)))}
	if m.VM && m.ThisMac && b.of.VM != "" {
		if len(s.Items) == 0 {
			s.Items = append(s.Items, page.Item{Empty: &page.Empty{Noun: "accounts", Text: "No sign-in on " + m.Name}})
		}
		if len(b.grantSheets()) > 0 {
			add := page.Row{ID: RowGrant, Label: GrantLabel, Mark: page.OK, Glyph: symbol("plus", page.SignalActive), Cells: []page.Value{page.Text(GrantStays)}, Trail: "value", Verb: ActionGrant}
			s.Items = append(s.Items, page.Item{Row: &add})
		}
	}
	return s, len(s.Items) > 0
}

// value: the sign-ins that need the owner: each program a machine is
// signed out of or must sign in to again, This Mac's first, with Sign in,
// which opens Terminal on the machine's accounts and waits while it runs.
func (b accountsBody) needs() ([]page.Item, []page.Action) {
	var items []page.Item
	var verbs []page.Action
	for _, m := range b.s.Machines {
		for _, r := range m.Rows {
			if r.Action != accounts.ActionSignIn && r.Action != accounts.ActionSignInAgain {
				continue
			}
			row, verb := b.signIn(m, r)
			items, verbs = append(items, page.Item{Row: &row}), append(verbs, verb)
		}
	}
	return items, verbs
}

func (b accountsBody) signIn(m MachineSignIns, r accounts.ProgramRow) (page.Row, page.Action) {
	key := rowOf(m.Name, r.Program)
	what := r.Program.Label() + " needs sign-in"
	mark, reason := page.Attention, page.Unset
	if r.State == accounts.ProgramBroken {
		mark, reason = page.Broken, page.Refused
	}
	row := page.Row{ID: key, Label: b.label(m), Detail: page.Text(what), Glyph: symbol("key", page.SignalWaits), Verb: ActionSignIn + key, Mark: mark, Reason: reason}
	if r.Line != "" {
		row.Detail = page.Texts(what, r.Line)
	}
	argv := []string{"accounts"}
	if m.VM || m.Name != b.self {
		argv = append(argv, "--on", m.Name)
	}
	verb := page.Action{ID: row.Verb, Label: VerbSignIn, On: page.OnRow, Rows: []string{key}, Busy: VerbWaiting, Default: true,
		Run: &page.Run{Terminal: true, Argv: argv}, Hud: hud("Terminal", "station accounts on "+b.label(m))}
	return row, verb
}

// value: Remove of a grant to a VM of This Mac, which only This Mac's
// vault can end; it asks as grants.remove does, in the engine's words.
func (b accountsBody) remove(m MachineSignIns, r accounts.ProgramRow) (page.Action, bool) {
	if r.How != accounts.HowGranted || r.Action != accounts.ActionRemove || r.Entry == "" || !m.ThisMac || !m.VM {
		return page.Action{}, false
	}
	key := rowOf(m.Name, r.Program)
	question := r.Ask
	if question == "" {
		question = VerbRemove + " " + r.Program.Label() + " from " + m.Name + "?"
	}
	levels := b.of.Asks
	if levels == nil {
		levels = yesOrNo
	}
	c := ask(levels, "remove-grant", machine.Workspace, m.Name, question, "remove")
	if c != nil {
		c.Danger = true
	}
	return page.Action{ID: ActionRemoveGrant + key, Label: VerbRemove, On: page.OnRow, Rows: []string{key}, Mutates: true, Confirm: c, Destructive: c != nil,
		Run: &page.Run{Call: CallRemoveGrant, In: answered(c, map[string]string{"Entry": r.Entry, "To": m.Name})},
		Hud: hud(r.Program.Label() + " removed from " + m.Name)}, true
}

func (b accountsBody) actions() []page.Action {
	_, out := b.needs()
	for _, m := range b.s.Machines {
		for _, r := range m.Rows {
			if remove, ok := b.remove(m, r); ok {
				out = append(out, remove)
			}
		}
	}
	if b.of.Mac != "" {
		out = append(out, b.grantSheets()...)
	}
	return out
}

// invariant: Grant a password: one sheet per machine it may go to, To
// swapping between them; This Mac's passwords as a pick list, one that
// cannot go there disabled with the engine's line; no term, since a
// grant stays until it is removed. The sheet the page opens is its first.
func (b accountsBody) grantSheets() []page.Action {
	to := b.grantTargets()
	if len(to) == 0 {
		return nil
	}
	if len(to) == 1 {
		return []page.Action{b.grantSheet(ActionGrant, to[0], nil)}
	}
	out := []page.Action{b.grantSheet(ActionGrant, to[0], to)}
	for _, t := range to {
		out = append(out, b.grantSheet(ActionGrant+":"+t, t, to))
	}
	return out
}

// value: the machines Grant a password offers: the page's own VM, else
// every machine of the page's Mac a password may go to, or of every Mac.
func (b accountsBody) grantTargets() []string {
	var out []string
	for _, t := range b.s.targets() {
		if b.of.VM != "" && t != b.of.VM {
			continue
		}
		if m, ok := b.s.machine(t); b.of.Mac != "" && (!ok || m.Mac != b.of.Mac) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (b accountsBody) grantSheet(id, vm string, every []string) page.Action {
	field := page.Field{Key: "To", Label: "To", Kind: page.FieldSelect, Swaps: true, Default: id}
	if len(every) == 0 {
		field.Options = []page.Option{toOption(id, vm)}
	}
	for _, o := range every {
		field.Options = append(field.Options, toOption(ActionGrant+":"+o, o))
	}
	if len(every) > 0 {
		field.Default = ActionGrant + ":" + vm
	}
	subtitle := "From the passwords of " + b.name(b.self) + ". It stays until you remove it."
	if p := b.s.presence(); p != "" {
		subtitle += " " + p + "."
	}
	form := &page.Form{Title: GrantLabel, Subtitle: subtitle, Fields: []page.Field{field, b.passwordsFor(vm)},
		Primary: []page.Primary{{When: &page.When{Key: "Entry", Set: true}, Label: "Grant {Entry}"}}}
	return page.Action{ID: id, Label: GrantLabel, On: page.OnRow, Rows: []string{RowGrant}, Mutates: true, Sheet: page.SheetGrant, Form: form,
		Run: &page.Run{Call: CallGrant, In: map[string]string{"To": vm}}, Hud: hud("Granted {Entry} to " + vm)}
}

func toOption(value, vm string) page.Option {
	return page.Option{Value: value, Label: vm, Glyph: page.Glyph{Kind: "workspace"}}
}

// value: This Mac's passwords for the machine: each with its program's
// logo and the engine's line there; one that cannot go there disabled
// with that line. The first that can is picked.
func (b accountsBody) passwordsFor(vm string) page.Field {
	f := page.Field{Key: "Entry", Label: "Password", Kind: page.FieldPick, Required: true}
	if b.s.Vault == nil {
		return f
	}
	for _, e := range b.s.Vault.Entries {
		fits, line := b.s.fit(e.ID, vm)
		o := page.Option{Value: e.ID, Label: e.Label, Note: line, Glyph: logo(e.Program)}
		if !fits {
			o.Disabled = line
		}
		f.Options = append(f.Options, o)
	}
	for _, o := range f.Options {
		if o.Disabled == "" {
			f.Default = o.Value
			break
		}
	}
	return f
}

// value: a VM's Accounts on its page: its sign-ins and grants, then Grant
// a password, with a footnote that says whose passwords they are.
func vmSignIns(m Machine, h model.MacView, vm model.VM) accountsBody {
	s := SignIns{}
	if m.SignIns != nil {
		s = *m.SignIns
	}
	return accountsBody{s: s, self: thisMacOf(*m.All), of: AccountsOf{Mac: h.Name, VM: vm.Name, Asks: m.Asks}}
}

func vmAccountsSection(m Machine, h model.MacView, vm model.VM) page.Section {
	b := vmSignIns(m, h, vm)
	b.s = b.s.within(h.Name, vm.Name)
	ms, ok := b.s.machine(vm.Name)
	if !ok {
		ms = MachineSignIns{Name: vm.Name, Mac: h.Name, ThisMac: h.ThisMac, VM: true}
	}
	s, _ := b.on(ms)
	s.ID, s.Title = SectionShared, "Accounts"
	note := "A grant is a password of " + b.name(b.self) + "'s, and stays until you remove it."
	if !h.ThisMac {
		note = "Grant from " + model.Label(*m.All, h) + ": its passwords are its own."
	}
	s.Items = append(s.Items, page.Item{Note: &page.Note{Text: note}})
	return s
}

func vmAccountActions(m Machine, h model.MacView, vm model.VM) []page.Action {
	b := vmSignIns(m, h, vm)
	b.s = b.s.within(h.Name, vm.Name)
	return b.actions()
}

// value: a Mac's Accounts segment: what needs you there about accounts,
// then each of its machines' sign-ins and grants; each section of the
// segment.
func macAccounts(m Machine, h model.MacView) ([]page.Section, []page.Action) {
	s := SignIns{}
	if m.SignIns != nil {
		s = m.SignIns.within(h.Name, "")
	}
	b := accountsBody{s: s, self: thisMacOf(*m.All), of: AccountsOf{Mac: h.Name, Asks: m.Asks}, segment: SegmentAccounts}
	return b.sections(), b.actions()
}

func grantedOn(m Machine, vm string) int {
	if m.SignIns == nil {
		return 0
	}
	ms, _ := m.SignIns.machine(vm)
	return SignIns{Machines: []MachineSignIns{ms}}.grants()
}

func thisMacOf(all model.AllMachines) string {
	for _, h := range all.Macs {
		if h.ThisMac {
			return h.Name
		}
	}
	return ""
}
