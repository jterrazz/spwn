package menubar

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

type workspace struct {
	// value: a running workspace of an awake Mac, a port may be shared
	// from and an account lent to.
	VM, Mac string
}

func runningWorkspaces(all model.AllMachines) []workspace {
	var out []workspace
	for _, h := range all.Macs {
		if h.Asleep || !h.ThisMac && !h.Reachable {
			continue
		}
		for _, vm := range liveFirst(h.VMs) {
			if vm.Lifetime != model.AlwaysOn && vm.State == model.Running {
				out = append(out, workspace{VM: vm.Name, Mac: h.Name})
			}
		}
	}
	return out
}

// value: a Mac's VMs, those whose agent finished after the others, so
// what a sheet picks first and a question names first is live work.
func liveFirst(vms []model.VM) []model.VM {
	out := slices.Clone(vms)
	slices.SortStableFunc(out, func(a, b model.VM) int { return finished(a) - finished(b) })
	return out
}

func finished(vm model.VM) int {
	if vm.Finished() {
		return 1
	}
	return 0
}

// value: the workspaces Lend an account… offers: the page's own VM, else
// the running workspaces of the page's Mac, or of every Mac.
func (b accountsBody) loanTargets() []workspace {
	if b.of.VM != "" {
		return []workspace{{VM: b.of.VM, Mac: b.of.Mac}}
	}
	var out []workspace
	if b.of.Machines == nil {
		return nil
	}
	for _, t := range runningWorkspaces(*b.of.Machines) {
		if b.of.Mac == "" || t.Mac == b.of.Mac {
			out = append(out, t)
		}
	}
	return out
}

// invariant: SPEC A5 Lend an account: one sheet per workspace it may go
// to, To swapping between them: the accounts of that workspace's Mac,
// the one signed out there offered disabled with its reason; For 4 hr, 12
// hr or 7 days, each saying when it ends by day; the button names the
// account and the time. The sheet the page opens is its first.
func (b accountsBody) loanSheets() []page.Action {
	to := b.loanTargets()
	if len(to) == 0 {
		mac := b.of.Mac
		if mac == "" {
			mac = b.self
		}
		return []page.Action{b.loanSheet(ActionShareAccount, workspace{Mac: mac}, nil)}
	}
	if len(to) == 1 {
		return []page.Action{b.loanSheet(ActionShareAccount, to[0], nil)}
	}
	out := []page.Action{b.loanSheet(ActionShareAccount, to[0], to)}
	for _, t := range to {
		out = append(out, b.loanSheet(ActionShareAccount+":"+t.VM, t, to))
	}
	return out
}

func (b accountsBody) loanSheet(id string, t workspace, every []workspace) page.Action {
	field := page.Field{Key: "To", Label: "To", Kind: page.FieldSelect, Swaps: true, Default: id}
	if len(every) == 0 && t.VM != "" {
		field.Options = []page.Option{b.toOption(id, t)}
	}
	for _, o := range every {
		field.Options = append(field.Options, b.toOption(ActionShareAccount+":"+o.VM, o))
		if o == t && id != ActionShareAccount {
			field.Default = ActionShareAccount + ":" + o.VM
		}
	}
	if len(every) > 0 && id == ActionShareAccount {
		field.Default = ActionShareAccount + ":" + t.VM
	}
	pick := b.accountsOf(t.Mac)
	form := &page.Form{Title: ShareLabel, Subtitle: "From " + b.name(t.Mac) + ". It ends by itself.",
		Fields:  []page.Field{field, pick, b.loanFor()},
		Primary: []page.Primary{{When: &page.When{Key: "Account", Set: true}, Label: "Lend {Account} for {For}"}}}
	a := page.Action{ID: id, Label: ShareLabel, On: page.OnRow, Rows: []string{RowShare}, Mutates: true, Sheet: page.SheetShareAccount, Form: form,
		Run: &page.Run{Call: CallShareAccount, In: map[string]string{"With": t.VM}}, Hud: hud("Shared {Account} with " + t.VM)}
	if t.Mac != "" && t.Mac != b.self {
		a.Run.Target = &page.Target{Name: t.Mac}
	}
	return a
}

func (b accountsBody) toOption(value string, t workspace) page.Option {
	return page.Option{Value: value, Label: t.VM, Note: "on " + b.name(t.Mac), Glyph: page.Glyph{Kind: "workspace"}}
}

// value: the accounts a Mac may share, as it is signed in to them: each
// with its logo and its words; one it is signed out of, disabled with
// why. The first it may share is picked.
func (b accountsBody) accountsOf(mac string) page.Field {
	f := page.Field{Key: "Account", Label: "Account", Kind: page.FieldPick, Required: true}
	r := b.hosts.Read()
	for _, on := range r.SignedIn {
		if on.Mac != mac {
			continue
		}
		for _, s := range on.Accounts {
			if _, ok := accounts.SecretOf(s.Provider); ok {
				f.Options = append(f.Options, page.Option{Value: s.Provider, Label: accounts.Label(s.Provider), Note: strings.Join(s.Words, "{sep}"), Glyph: logo(s.Provider)})
			}
		}
	}
	for _, s := range r.SignedOut {
		if _, ok := accounts.SecretOf(s.Provider); ok && s.Mac == mac {
			f.Options = append(f.Options, page.Option{Value: s.Provider, Label: accounts.Label(s.Provider), Note: "needs sign-in on " + b.name(mac),
				Disabled: "needs sign-in on " + b.name(mac), Glyph: logo(s.Provider)})
		}
	}
	for _, o := range f.Options {
		if o.Disabled == "" {
			f.Default = o.Value
			break
		}
	}
	return f
}

// value: For: 4 hr, 12 hr or 7 days, each saying when it ends by day.
func (b accountsBody) loanFor() page.Field {
	f := page.Field{Key: "For", Label: "For", Kind: page.FieldSegmented, Options: shareFor()}
	f.Default = f.Options[0].Value
	for i, o := range f.Options {
		n, _ := strconv.ParseInt(o.Value, 10, 64)
		f.Options[i].Note = endsBy(b.now.Now, b.now.Now.Add(time.Duration(n)))
	}
	return f
}

func endsBy(now, end time.Time) string {
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	switch {
	case end.Before(today.AddDate(0, 0, 1)):
		return "Ends at " + end.Format("15:04") + ", then taken back"
	case end.Before(today.AddDate(0, 0, 2)):
		return "Ends tomorrow at " + end.Format("15:04")
	}
	return "Ends " + end.Format("Monday 2 Jan")
}
