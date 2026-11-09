package menubar

import (
	"github.com/jterrazz/studio-station/src/accounts"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	"strings"
)

// value: a Mac's Accounts segment: what needs you there about accounts,
// the copies it lent to its VMs with Lend an account… under them, then
// what it is signed in to; each section of the segment.
func macAccounts(m Machine, h model.MacView) ([]page.Section, []page.Action) {
	all := accounts.AllMachines{Now: m.Now}
	if m.Shares != nil {
		all = m.Shares.Of(h.Name, "")
	}
	b := accountsBody{r: all.Read(), now: all, self: thisMacOf(*m.All), of: AccountsOf{Mac: h.Name}, segment: SegmentAccounts}
	return b.sections(), b.actions()
}

// value: a workspace's Accounts segment: the copies its Mac shared it, each
// with its time left, Extend and Take back, then Lend an account…; with
// none, the line that says so; the footnote names whose accounts they are.
func sharedSection(m Machine, h model.MacView, vm model.VM) page.Section {
	b := vmAccounts(m, h, vm)
	s := b.copies()
	said := map[string][]string{}
	for _, a := range sharedAccounts(m, vm.Name) {
		for _, scope := range a.Scopes {
			if !strings.Contains(scope, ":") {
				said[a.Provider] = append(said[a.Provider], scope)
			}
		}
	}
	for i, it := range s.Items {
		if it.Row == nil || it.Row.ID == RowShare {
			continue
		}
		provider := providerOf(*it.Row)
		s.Items[i].Row.Menu = append(s.Items[i].Row.Menu, ActionTakeBack+"/"+provider)
		if words := said[provider]; len(words) > 0 {
			s.Items[i].Row.Detail = page.Texts(words...)
		}
	}
	s.Items = append(s.Items, page.Item{Note: &page.Note{Text: "These are " + model.Label(*m.All, h) + "'s accounts. Every loan ends by itself."}})
	return s
}

func vmAccounts(m Machine, h model.MacView, vm model.VM) accountsBody {
	mac := accounts.MacAccounts{Name: h.Name, ThisMac: h.ThisMac}
	for _, a := range sharedAccounts(m, vm.Name) {
		mac.Shares = append(mac.Shares, accounts.Shared{Account: a.Provider, VM: vm.Name, At: a.Shared.At, Until: a.Shared.Until})
	}
	all := accounts.AllMachines{Now: m.Now, Macs: []accounts.MacAccounts{mac}}
	return accountsBody{r: all.Read(), now: all, self: thisMacOf(*m.All), of: AccountsOf{Mac: h.Name, VM: vm.Name}}
}

// value: the actions of a workspace's Accounts segment: Extend of each
// copy and Lend an account…, beside the Take backs.
func vmAccountActions(m Machine, h model.MacView, vm model.VM) []page.Action {
	b := vmAccounts(m, h, vm)
	out := make([]page.Action, 0, len(b.r.Shared)+1)
	for _, c := range b.r.Shared {
		out = append(out, b.extend(c))
	}
	return append(out, b.share(vm.Name))
}

func providerOf(r page.Row) string {
	for i := len(r.ID) - 1; i >= 0; i-- {
		if r.ID[i] == '/' {
			return r.ID[i+1:]
		}
	}
	return r.ID
}

func sharedAccounts(m Machine, vm string) []accounts.Account {
	if m.Accounts == nil {
		return nil
	}
	var out []accounts.Account
	for _, mc := range m.Accounts.Machines {
		if mc.Name != vm {
			continue
		}
		for _, a := range mc.Accounts {
			if a.Shared != nil {
				out = append(out, a)
			}
		}
	}
	return out
}

func takeBacks(m Machine, h model.MacView, vm model.VM) []page.Action {
	shared := sharedAccounts(m, vm.Name)
	out := make([]page.Action, 0, len(shared))
	for _, a := range shared {
		label := accounts.Label(a.Provider)
		out = append(out, takeBack(label, vm.Name, model.Label(*m.All, h)).action(m.Asks, page.Action{ID: ActionTakeBack + "/" + a.Provider, Label: "Take back", On: page.OnRow,
			Rows: []string{h.Name + "/" + vm.Name + "/" + a.Provider}, Priority: 4, Mutates: true,
			Run: &page.Run{Call: CallTakeBack, In: map[string]string{"Account": a.Provider, "From": vm.Name}, Target: target(h)}}))
	}
	return append(out, vmAccountActions(m, h, vm)...)
}

func thisMacOf(all model.AllMachines) string {
	for _, h := range all.Macs {
		if h.ThisMac {
			return h.Name
		}
	}
	return ""
}
