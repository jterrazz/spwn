package menubar

import (
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
	pageaccounts "github.com/jterrazz/studio-station/src/pages/accounts"
)

const (
	AccountsID page.ID = "menubar.accounts"

	SectionNeedsYou    = "needs-you"
	SectionSignedIn    = "signed-in:"
	SectionSignedInAll = "signed-in"
	SectionSilent      = "silent"

	RowShare = "share-account"

	ActionShareAccount = "share-account"
	ActionSignIn       = "sign-in:"
	ActionExtend       = "extend:"
	VerbWaiting        = "Waiting"
	ShareLabel         = "Lend an account"
	ShareEnds          = "ends by itself"
	FullListInTerminal = "Full list in Terminal"
)

// value: how long a share lasts, as the sheet offers it: 4 hr, 12 hr, 7
// days, each sent as nanoseconds.
func shareFor() []page.Option {
	return []page.Option{
		{Value: strconv.FormatInt(int64(4*time.Hour), 10), Label: "4 hr"},
		{Value: strconv.FormatInt(int64(12*time.Hour), 10), Label: "12 hr"},
		{Value: strconv.FormatInt(int64(7*24*time.Hour), 10), Label: "7 days"},
	}
}

type AccountsOf struct {
	// value: the Mac whose accounts the page lists, and with VM the copies
	// that Mac shared with it; both empty is every Mac.
	Mac, VM string
	// value: the level each gesture asks at, the calls' own rule.
	Asks Levels
}

func Accounts(a accounts.AllMachines, of AccountsOf, thisMac page.Context) page.Page {
	// invariant: SPEC A4, B5.8: Accounts · All machines: its header and
	// facts (the accounts signed in, the copies shared); what needs you first,
	// the sign-ins a Mac is out of and the copies with less than 30 min
	// left, each with its verb; every copy lent to a VM with its time left
	// and its total, the one ending first first; then each account signed
	// in somewhere with the Macs that hold it (one Mac's page: that Mac's
	// sign-ins in its provider's words); the footer lists them all in
	// Terminal and shares one.
	if of.Mac != "" {
		a = a.Of(of.Mac, of.VM)
	}
	b := accountsBody{r: a.Read(), now: a, self: thisMac.Machine, of: of}
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
	if of.VM == "" {
		share := b.share("")
		share.On, share.Rows, share.Icon = page.OnPage, nil, "plus"
		p.Actions = append(p.Actions, share)
		p.Footer.Buttons = append(p.Footer.Buttons, share.ID)
	}
	for _, c := range b.r.Shared {
		withRowMenu(&p, c.Mac+"/"+c.VM+"/"+c.Provider, leaseMenu(c, b.extend(c), of.Asks, b.self))
	}
	return page.Seal(p)
}

type accountsBody struct {
	r    accounts.Reading
	now  accounts.AllMachines
	self string
	of   AccountsOf
	// value: the segment of a machine page the sections belong to; empty
	// on the fleet page.
	segment string
}

func (b accountsBody) name(mac string) string { return page.MacName(mac, b.self) }

func (b accountsBody) id(s string) string {
	if b.segment == "" {
		return s
	}
	return b.segment + ":" + s
}

// value: the page's facts: where, the accounts signed in (one per
// provider, however many Macs hold it), the copies shared.
func (b accountsBody) facts(where string) []page.Value {
	var providers []string
	for _, on := range b.r.SignedIn {
		for _, s := range on.Accounts {
			if !slices.Contains(providers, s.Provider) {
				providers = append(providers, s.Provider)
			}
		}
	}
	return []page.Value{page.Text(where), page.Text(counted(len(providers), "account", "accounts")), page.Text(strconv.Itoa(len(b.r.Shared)) + " lent to VMs")}
}

func (b accountsBody) sections() []page.Section {
	var out []page.Section
	if needs, _ := b.needs(); len(needs) > 0 {
		count := page.Text(strconv.Itoa(len(needs)))
		count.Signal = page.SignalWaits
		out = append(out, page.Section{ID: b.id(SectionNeedsYou), Title: "Needs you", Segment: b.segment, Card: page.CardFlush, Uncounted: true, Columns: columns(), Caption: []page.Value{count}, Items: needs})
	}
	if copies := b.copies(); len(copies.Items) > 0 {
		out = append(out, copies)
	}
	if b.of.Mac == "" {
		if s, ok := b.signedInAnywhere(); ok {
			out = append(out, s)
		}
	}
	for _, on := range b.r.SignedIn {
		if b.of.Mac == "" {
			break
		}
		items := make([]page.Item, 0, len(on.Accounts))
		for _, s := range on.Accounts {
			row := page.Row{ID: s.Mac + "/" + s.Provider, Label: accounts.Label(s.Provider), Mark: page.OK, Glyph: logo(s.Provider)}
			if len(s.Words) > 0 {
				row.Detail = page.Texts(s.Words...)
			}
			items = append(items, page.Item{Row: &row})
		}
		out = append(out, page.Section{ID: b.id(SectionSignedIn + on.Mac), Title: "Signed in on " + b.name(on.Mac), Segment: b.segment, Card: page.CardFlush, Columns: columns(),
			Caption: []page.Value{page.Text(strconv.Itoa(len(items)))}, Items: items})
	}
	silent := make([]page.Item, 0, len(b.r.Silent))
	for _, s := range b.r.Silent {
		row := page.Row{ID: s.Mac, Label: s.Mac, Short: b.name(s.Mac), Mark: page.NotKnown, Reason: page.Unreachable, Cells: []page.Value{page.Text(s.Why)}}
		silent = append(silent, page.Item{Row: &row})
	}
	if len(silent) > 0 {
		out = append(out, page.Section{ID: b.id(SectionSilent), Title: "Not read", Segment: b.segment, Uncounted: true, Columns: columns(), Items: silent})
	}
	if len(out) == 0 {
		out = []page.Section{{ID: b.id(SectionSignedIn), Title: "Accounts", Segment: b.segment, Columns: columns(), Items: []page.Item{{Empty: &page.Empty{Noun: "accounts"}}}}}
	}
	return out
}

// value: the fleet's sign-ins: one row per account, the Macs signed in
// to it under it, This Mac's first; the account most Macs hold first.
func (b accountsBody) signedInAnywhere() (page.Section, bool) {
	var order []string
	on := map[string][]string{}
	for _, m := range b.r.SignedIn {
		for _, s := range m.Accounts {
			if _, ok := on[s.Provider]; !ok {
				order = append(order, s.Provider)
			}
			on[s.Provider] = append(on[s.Provider], b.name(s.Mac))
		}
	}
	slices.SortStableFunc(order, func(x, y string) int { return len(on[y]) - len(on[x]) })
	s := page.Section{ID: SectionSignedInAll, Title: "Signed in", Card: page.CardFlush, Columns: columns(), Caption: []page.Value{page.Text(strconv.Itoa(len(order)))}}
	for _, provider := range order {
		row := page.Row{ID: provider, Label: accounts.Label(provider), Mark: page.OK, Glyph: logo(provider), Detail: page.Texts(on[provider]...)}
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	return s, len(order) > 0
}

// value: the copies shared, each with its time left over its total; on a
// Mac's page and a VM's, Lend an account… under them (with no copy to a
// VM, the line that says so first), since the fleet page shares from its
// footer.
func (b accountsBody) copies() page.Section {
	title := "Lent to VMs"
	if b.of.VM != "" {
		title = "Lent to " + b.of.VM
	}
	s := page.Section{ID: b.id(SectionShared), Title: title, Segment: b.segment, Card: page.CardFlush, Columns: columns(), Caption: []page.Value{page.Text(strconv.Itoa(len(b.r.Shared)))}}
	for _, c := range b.r.Shared {
		row := sharedRow(c)
		row.Menu = []string{b.extend(c).ID}
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	if b.of.Mac == "" {
		return s
	}
	if len(s.Items) == 0 && b.of.VM != "" {
		s.Items = append(s.Items, page.Item{Empty: &page.Empty{Noun: "accounts", Text: "No account lent to " + b.of.VM}})
	}
	add := page.Row{ID: RowShare, Label: ShareLabel, Mark: page.OK, Glyph: symbol("plus", page.SignalActive), Cells: []page.Value{page.Text(ShareEnds)}, Trail: "value", Verb: ActionShareAccount}
	s.Items = append(s.Items, page.Item{Row: &add})
	return s
}

// value: the account matters that need the owner, ranked as Home ranks
// them (SPEC B5.7, B5.8): a copy ending under 30 min, soonest first, with
// Extend; then each sign-in a Mac is out of, This Mac's first, with Sign
// in, which opens Terminal on the login and waits while it runs.
func (b accountsBody) needs() ([]page.Item, []page.Action) {
	w := needSaid{r: NeedsReading{Now: b.now.Now}, self: b.self}
	var items []page.Item
	var verbs []page.Action
	for _, c := range b.r.Shared {
		if !c.Short {
			continue
		}
		n := w.ending(c)
		key := "short/" + c.Mac + "/" + c.VM + "/" + c.Provider
		n.Verb.Rows = []string{key}
		row := page.Row{ID: key, Label: n.Subject, Detail: page.Text(n.What), Glyph: n.Glyph, Verb: n.Verb.ID, Mark: page.Attention, Reason: page.Expiring}
		items, verbs = append(items, page.Item{Row: &row}), append(verbs, n.Verb)
	}
	for _, s := range b.r.SignedOut {
		row, verb := b.signIn(s)
		items, verbs = append(items, page.Item{Row: &row}), append(verbs, verb)
	}
	return items, verbs
}

func (b accountsBody) signIn(s accounts.SignIn) (page.Row, page.Action) {
	key := s.Mac + "/" + s.Provider
	said := "station accounts"
	if s.Hint != "" {
		said = s.Hint
	}
	row := page.Row{ID: key, Label: b.name(s.Mac), Detail: page.Text(accounts.Label(s.Provider) + " needs sign-in"), Glyph: symbol("key", page.SignalWaits), Verb: ActionSignIn + key,
		Mark: pages.MarkOf(s.State), Reason: pageaccounts.Reason(s.Reason)}
	if row.Mark == page.Absent {
		row.Mark = page.Attention
	}
	if row.Reason == "" {
		row.Reason = page.Unset
	}
	if s.Hint != "" {
		row.Hint = page.Cmd(s.Hint)
	}
	verb := page.Action{ID: row.Verb, Label: VerbSignIn, On: page.OnRow, Rows: []string{key}, Busy: VerbWaiting, Default: true,
		Run: &page.Run{Terminal: true, Argv: onMac([]string{"accounts"}, s.Mac, b.self)}, Hud: hud("Terminal", said+" on "+b.name(s.Mac))}
	return row, verb
}

func (b accountsBody) extend(c accounts.SharedCopy) page.Action {
	key := c.Mac + "/" + c.VM + "/" + c.Provider
	account := accounts.Label(c.Provider)
	run := &page.Run{Call: CallShareAccount, In: map[string]string{"Account": c.Provider, "With": c.VM, "For": strconv.FormatInt(int64(extendBy), 10)}}
	if c.Mac != b.self {
		run.Target = &page.Target{Name: c.Mac}
	}
	return page.Action{ID: ActionExtend + key, Label: VerbExtend, On: page.OnRow, Rows: []string{key}, Mutates: true, Run: run,
		Hud: hud(account+" on "+c.VM, minutes(extendBy)+" left")}
}

// value: Lend an account…: the sheet that shares one of the Mac's accounts
// for a time, to the VM named when there is one.
func (b accountsBody) share(vm string) page.Action {
	from := b.self
	if b.of.Mac != "" {
		from = b.of.Mac
	}
	form := &page.Form{Title: ShareLabel, Subtitle: "From " + b.name(from) + ". It ends by itself.", Fields: []page.Field{
		{Key: "With", Label: "To", Kind: page.FieldText, Required: true, Default: vm},
		{Key: "Account", Label: "Account", Kind: page.FieldText, Required: true},
		{Key: "For", Label: "For", Kind: page.FieldSegmented, Default: shareFor()[0].Value, Options: shareFor()},
	}}
	a := page.Action{ID: ActionShareAccount, Label: ShareLabel, On: page.OnRow, Rows: []string{RowShare}, Mutates: true, Sheet: page.SheetShareAccount, Form: form,
		Run: &page.Run{Call: CallShareAccount, In: map[string]string{}}}
	if vm != "" {
		a.Prefill = map[string]string{"With": vm}
	}
	if b.of.Mac != "" && b.of.Mac != b.self {
		a.Run.Target = &page.Target{Name: b.of.Mac}
	}
	return a
}

func (b accountsBody) actions() []page.Action {
	_, out := b.needs()
	for _, c := range b.r.Shared {
		out = append(out, b.extend(c))
	}
	if b.of.Mac != "" {
		out = append(out, b.share(b.of.VM))
	}
	return out
}

func sharedRow(c accounts.SharedCopy) page.Row {
	row := page.Row{ID: c.Mac + "/" + c.VM + "/" + c.Provider, Label: accounts.Label(c.Provider), Mac: c.VM, Mark: page.OK, Glyph: logo(c.Provider), Cells: []page.Value{page.Text(c.VM)}, Left: &page.Value{Kind: page.KindUntil, N: int64(c.Left.Seconds()), Of: int64(c.Total.Seconds())}}
	if c.Scope != "" {
		row.Detail = page.Text(c.Scope)
	}
	if c.Short {
		row.Mark, row.Reason = page.Attention, page.Expiring
	}
	return row
}

func logo(provider string) page.Glyph { return page.Glyph{Kind: "account:" + provider} }
