package accounts

import (
	"strings"
	"time"

	model "github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/finding"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
)

func stateWord(a model.Account) string {
	switch {
	case a.State == catalog.StateOK:
		return "signed in"
	case a.State == catalog.StateBroken && a.Reason == model.ReasonRefused:
		return "expired"
	case a.State == catalog.StateAbsent:
		return "not signed in"
	case a.State == catalog.StatePending:
		return "reading"
	case a.Reason == model.ReasonLate:
		return "late"
	}
	return string(mark(a))
}

func who(a model.Account) string {
	if len(a.Identity) == 1 && a.Identity[0] == model.Unknown {
		return ""
	}
	return strings.Join(a.Identity, ", ")
}

func kindWord(kind string) string {
	if kind == model.KindVM {
		return "workspace"
	}
	return "computer"
}

func ownRecords(m model.Machine) []page.Item {
	var items []page.Item
	for _, a := range m.Accounts {
		if a.Shared != nil && a.Origin == model.OriginShared {
			continue
		}
		rec := page.Row{ID: model.RowID(m.Name, a.Provider), Label: a.Provider, Mark: mark(a)}
		rec.Cells = []page.Value{page.Text(stateWord(a)), page.Text(who(a)), page.Texts(a.Scopes...), page.Text(a.Detail), page.Text(a.Hint)}
		if rec.Mark == page.Attention || rec.Mark == page.Broken {
			rec.Reason = Reason(a.Reason)
		}
		items = append(items, page.Item{Row: &rec})
	}
	return items
}

func SharesSection(shared []model.Shared, now time.Time) page.Section {
	// invariant: every share is listed with its VM and its time left; one in
	// its last hour is expiring with the share that renews it, an ended one
	// the engine could not take back is failing with its take-back; a Mac
	// that shares nothing says so in one line.
	s := page.Section{ID: "shares", Title: "Shared with VMs", Columns: []page.Column{
		{Key: "account", Kind: page.KindText}, {Key: "vm", Kind: page.KindText}, {Key: "left", Kind: page.KindText},
	}}
	for _, sh := range shared {
		r := page.Row{ID: "share/" + sh.VM + "/" + sh.Account, Label: sh.Account, Short: model.Label(sh.Account), Mark: page.OK,
			Cells: []page.Value{page.Text(sh.VM), page.Text(left(sh.Until, now))}}
		switch {
		case !now.Before(sh.Until):
			r.Mark, r.Reason = page.Broken, page.Failing
			r.Hint = &page.Hint{Command: "station accounts take-back " + sh.Account + " --from " + sh.VM}
		case sh.Until.Sub(now) <= time.Hour:
			r.Mark, r.Reason = page.Attention, page.Expiring
			r.Hint = &page.Hint{Command: "station accounts share " + sh.Account + " --with " + sh.VM}
		}
		s.Items = append(s.Items, page.Item{Row: &r})
	}
	if len(s.Items) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Noun: "shares"}}}
	}
	return s
}

func List(r model.Report, shared []model.Shared) page.Page {
	// value: the Accounts section as the CLI prints it piped: the machine's
	// own sign-ins, one row per provider, then the shares it holds or makes.
	r = r.Resolved()
	s := page.Section{ID: "records", Title: "Accounts", Role: page.Banner, Columns: []page.Column{
		{Key: "provider", Kind: page.KindText}, {Key: "state", Kind: page.KindText}, {Key: "who", Kind: page.KindText},
		{Key: "scope", Kind: page.KindText}, {Key: "detail", Kind: page.KindText}, {Key: "hint", Kind: page.KindText},
	}}
	if len(r.Machines) > 0 {
		s.Items = ownRecords(r.Machines[0])
	}
	if len(s.Items) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Noun: "accounts"}}}
	}
	return page.Seal(page.Page{ID: "accounts", Command: "accounts", Sections: []page.Section{s, SharesSection(shared, r.Now)}})
}

type MachineBlock struct {
	// value: one machine of --all-machines: its reading, the shares it
	// holds, or the line that says why it has none.
	Name   string
	Report model.Report
	Shared []model.Shared
	Note   string
	// value: a machine not read, marked as every surface marks it: a
	// stopped VM, a Mac that does not answer; State is its word.
	Mark   page.Mark
	Reason page.Reason
	State  string
	// value: how its document names it (SPEC §11.1), and why it was not
	// read: the type and code of the error it answered, or that it is a server.
	Label, Kind, On string
	ThisMac         bool
	Fault           Fault
	Server          bool
}

type Fault struct{ Type, Code string }

func EveryMachine(blocks []MachineBlock) page.Page {
	var sections []page.Section
	for _, b := range blocks {
		if b.Mark != "" {
			r := page.Row{ID: b.Name, Label: b.Name, Mark: b.Mark, Reason: b.Reason, Cells: []page.Value{page.Text(b.State)}}
			sections = append(sections, page.Section{ID: b.Name, Title: b.Name, Items: []page.Item{{Row: &r}}})
			continue
		}
		if b.Note != "" {
			sections = append(sections, page.Section{ID: b.Name, Title: b.Name, Items: []page.Item{{Note: &page.Note{Text: b.Note}}}})
			continue
		}
		one := List(b.Report, b.Shared)
		own, shares := one.Sections[0], one.Sections[1]
		own.ID, own.Title, own.Role = b.Name, b.Name, ""
		shares.ID = b.Name + "/shares"
		sections = append(sections, own, shares)
	}
	return page.Seal(page.Page{ID: "accounts", Command: "accounts", Sections: sections})
}

type ListJSON struct {
	Schema    string         `json:"schema"`
	Machine   MachineRef     `json:"machine"`
	Findings  []finding.JSON `json:"findings"`
	Accounts  []AccountRow   `json:"accounts"`
	Shares    []ShareRow     `json:"shares"`
	ShareList []string       `json:"share_list"`
}

type EveryJSON struct {
	Schema   string         `json:"schema"`
	Findings []finding.JSON `json:"findings"`
	Machines []MachineJSON  `json:"machines"`
	Skipped  []SkippedJSON  `json:"skipped"`
}

type MachineJSON struct {
	Machine   MachineName  `json:"machine"`
	Accounts  []AccountRow `json:"accounts"`
	Shares    []ShareRow   `json:"shares"`
	ShareList []string     `json:"share_list"`
	Error     *ErrorJSON   `json:"error"`
}

type MachineName struct {
	Name    string  `json:"name"`
	Label   string  `json:"label"`
	Kind    string  `json:"kind"`
	On      *string `json:"on"`
	ThisMac bool    `json:"this_mac"`
}

type ErrorJSON struct {
	Type    string  `json:"type"`
	Code    *string `json:"code"`
	Message string  `json:"message"`
}

type SkippedJSON struct {
	Name string `json:"name"`
	Why  string `json:"why"`
}

type MachineRef struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type AccountRow struct {
	Name      string `json:"name"`
	Label     string `json:"label"`
	State     string `json:"state"`
	Who       string `json:"who,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Detail    string `json:"detail,omitempty"`
	Fix       string `json:"fix,omitempty"`
	Shareable bool   `json:"shareable"`
}

type ShareRow struct {
	Account string    `json:"account"`
	VM      string    `json:"vm"`
	At      time.Time `json:"at"`
	UntilAt time.Time `json:"until_at"`
}

func ListContract(schema string, r model.Report, shared []model.Shared) ListJSON {
	// invariant: names, states, who and scopes only: no field of it ever
	// carries a token, a cookie or a key body.
	r = r.Resolved()
	out := ListJSON{Schema: schema, Findings: finding.Doc(model.Findings(r, machine.Ref{})), Accounts: []AccountRow{}, Shares: []ShareRow{}, ShareList: r.ShareList}
	if out.ShareList == nil {
		out.ShareList = []string{}
	}
	if len(r.Machines) > 0 {
		m := r.Machines[0]
		out.Machine = MachineRef{Name: m.Name, Kind: kindWord(m.Kind)}
		for _, a := range m.Accounts {
			if a.Shared != nil && a.Origin == model.OriginShared {
				continue
			}
			secret, ok := model.SecretOf(a.Provider)
			out.Accounts = append(out.Accounts, AccountRow{
				Name: a.Provider, Label: model.Label(a.Provider), State: strings.ReplaceAll(stateWord(a), " ", "-"),
				Who: who(a), Scope: strings.Join(a.Scopes, ", "), Detail: a.Detail, Fix: a.Hint,
				Shareable: ok && model.Shareable(secret, r.ShareList),
			})
		}
	}
	for _, sh := range shared {
		out.Shares = append(out.Shares, ShareRow{Account: sh.Account, VM: sh.VM, At: sh.At, UntilAt: sh.Until})
	}
	return out
}

func EveryMachineContract(schema string, blocks []MachineBlock) EveryJSON {
	// invariant: one entry per Mac and workspace read, in the order of the
	// page; one that did not answer keeps its place with its error, a
	// server is skipped and said (SPEC §11.1).
	out := EveryJSON{Schema: schema, Machines: []MachineJSON{}, Skipped: []SkippedJSON{}}
	var fs []finding.Finding
	for _, b := range blocks {
		if b.Server {
			out.Skipped = append(out.Skipped, SkippedJSON{Name: b.Name, Why: "server"})
			continue
		}
		name := MachineName{Name: b.Name, Label: b.Label, Kind: b.Kind, ThisMac: b.ThisMac}
		if b.On != "" {
			on := b.On
			name.On = &on
		}
		m := MachineJSON{Machine: name, Accounts: []AccountRow{}, Shares: []ShareRow{}, ShareList: []string{}}
		if b.Mark != "" || b.Note != "" {
			e := ErrorJSON{Type: b.Fault.Type, Message: b.Note}
			if b.Fault.Code != "" {
				code := b.Fault.Code
				e.Code = &code
			}
			if e.Message == "" {
				e.Message = b.State
			}
			m.Error = &e
			out.Machines = append(out.Machines, m)
			continue
		}
		one := ListContract(schema, b.Report, b.Shared)
		m.Accounts, m.Shares, m.ShareList = one.Accounts, one.Shares, one.ShareList
		out.Machines = append(out.Machines, m)
		fs = append(fs, model.Findings(b.Report, machine.Ref{Name: b.Name, On: b.On})...)
	}
	out.Findings = finding.AllMachinesDoc(fs)
	return out
}

func Shared(sh model.Shared, now time.Time) page.Page {
	r := page.Row{ID: "share/" + sh.VM + "/" + sh.Account, Label: sh.Account, Short: model.Label(sh.Account), Mark: page.OK,
		Cells: []page.Value{page.Text("shared with " + sh.VM), page.Text(model.Left(sh.Until, now))}}
	return page.Seal(page.Page{ID: "outcome-accounts-share", Command: "accounts share", Sections: []page.Section{{ID: "outcome", Title: "Shared", Role: page.Banner, Columns: []page.Column{
		{Key: "what", Kind: page.KindText}, {Key: "left", Kind: page.KindText},
	}, Items: []page.Item{{Row: &r}}}}})
}

func TakenBack(sh model.Shared) page.Page {
	r := page.Row{ID: "share/" + sh.VM + "/" + sh.Account, Label: sh.Account, Short: model.Label(sh.Account), Mark: page.OK,
		Cells: []page.Value{page.Text("taken back from " + sh.VM)}}
	return page.Seal(page.Page{ID: "outcome-accounts-take-back", Command: "accounts take-back", Sections: []page.Section{{ID: "outcome", Title: "Taken back", Role: page.Banner, Columns: []page.Column{
		{Key: "what", Kind: page.KindText},
	}, Items: []page.Item{{Row: &r}}}}})
}

func left(until, now time.Time) string {
	if !now.Before(until) {
		return "ended"
	}
	return model.Left(until, now) + " left"
}
