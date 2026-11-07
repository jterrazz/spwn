package menubar

import (
	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/pages"
	pageaccounts "github.com/jterrazz/studio-station/src/pages/accounts"
)

const (
	AccountsID page.ID = "menubar.accounts"

	SectionNeedsYou = "needs-you"
	SectionSignedIn = "signed-in:"
	SectionSilent   = "silent"
)

type AccountsOf struct {
	// value: the Mac whose accounts the page lists, and with VM the copies
	// that Mac shared with it; both empty is every Mac.
	Mac, VM string
}

func Accounts(a accounts.AllMachines, of AccountsOf, thisMac page.Context) page.Page {
	// invariant: SPEC B5.8: what needs you first, the sign-ins a Mac is out
	// of and the copies with less than 30 min left; then each Mac's sign-ins
	// in its provider's words; then every copy shared with a VM with its
	// time left and its total.
	if of.Mac != "" {
		a = a.Of(of.Mac, of.VM)
	}
	r := a.Read()
	name := func(mac string) string { return page.MacName(mac, thisMac.Machine) }
	p := page.Page{ID: AccountsID, Command: "accounts", Context: thisMac}
	var needs []page.Item
	for _, s := range r.SignedOut {
		row := page.Row{ID: s.Mac + "/" + s.Provider, Label: accounts.Label(s.Provider), Mac: name(s.Mac), Mark: pages.MarkOf(s.State), Reason: pageaccounts.Reason(s.Reason), Glyph: logo(s.Provider), Cells: []page.Value{page.Text("Signed out on " + name(s.Mac))}}
		if row.Mark == page.Absent {
			row.Mark = page.Attention
		}
		if row.Reason == "" {
			row.Reason = page.Unset
		}
		if s.Hint != "" {
			row.Hint = page.Cmd(s.Hint)
		}
		needs = append(needs, page.Item{Row: &row})
	}
	shared := make([]page.Item, 0, len(r.Shared))
	for _, c := range r.Shared {
		row := sharedRow(c, name)
		if c.Short {
			short := row
			short.ID = "short/" + row.ID
			needs = append(needs, page.Item{Row: &short})
		}
		shared = append(shared, page.Item{Row: &row})
	}
	if len(needs) > 0 {
		p.Sections = append(p.Sections, page.Section{ID: SectionNeedsYou, Title: "Needs you", Columns: columns(), Items: needs})
	}
	for _, on := range r.SignedIn {
		var items []page.Item
		for _, s := range on.Accounts {
			row := page.Row{ID: s.Mac + "/" + s.Provider, Label: accounts.Label(s.Provider), Mac: name(s.Mac), Mark: page.OK, Glyph: logo(s.Provider)}
			if len(s.Words) > 0 {
				row.Detail = page.Texts(s.Words...)
			}
			items = append(items, page.Item{Row: &row})
		}
		p.Sections = append(p.Sections, page.Section{ID: SectionSignedIn + on.Mac, Title: "Signed in on " + name(on.Mac), Columns: columns(), Items: items})
	}
	if len(shared) > 0 || of.VM != "" {
		title := "Shared with VMs"
		if of.VM != "" {
			title = "Shared with " + of.VM
		}
		s := page.Section{ID: SectionShared, Title: title, Columns: columns(), Items: shared}
		if len(shared) == 0 {
			s.Items = []page.Item{{Empty: &page.Empty{Noun: "accounts", Text: "No account shared with " + of.VM}}}
		}
		p.Sections = append(p.Sections, s)
	}
	silent := make([]page.Item, 0, len(r.Silent))
	for _, s := range r.Silent {
		row := page.Row{ID: s.Mac, Label: s.Mac, Short: name(s.Mac), Mark: page.NotKnown, Reason: page.Unreachable, Cells: []page.Value{page.Text(s.Why)}}
		silent = append(silent, page.Item{Row: &row})
	}
	if len(silent) > 0 {
		p.Sections = append(p.Sections, page.Section{ID: SectionSilent, Title: "Not read", Uncounted: true, Columns: columns(), Items: silent})
	}
	if len(p.Sections) == 0 {
		p.Sections = []page.Section{{ID: SectionSignedIn, Title: "Accounts", Columns: columns(), Items: []page.Item{{Empty: &page.Empty{Noun: "accounts"}}}}}
	}
	return page.Seal(p)
}

func sharedRow(c accounts.SharedCopy, name func(string) string) page.Row {
	row := page.Row{ID: c.Mac + "/" + c.VM + "/" + c.Provider, Label: accounts.Label(c.Provider), Mac: name(c.Mac), Mark: page.OK, Glyph: logo(c.Provider), Cells: []page.Value{page.Text(c.VM)}, Left: &page.Value{Kind: page.KindUntil, N: int64(c.Left.Seconds()), Of: int64(c.Total.Seconds())}}
	if c.Scope != "" {
		row.Detail = page.Text(c.Scope)
	}
	if c.Short {
		row.Mark, row.Reason = page.Attention, page.Expiring
	}
	return row
}

func logo(provider string) page.Glyph { return page.Glyph{Kind: "account:" + provider} }
