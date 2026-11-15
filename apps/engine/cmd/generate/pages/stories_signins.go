package pages

import (
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemenubar "github.com/jterrazz/studio-station/apps/engine/pages/menubar"
)

// value: the scene's sign-ins and grants as the engine's feed reads them:
// each reachable Mac and each running VM with the rows accounts.list gives
// it, This Mac's vault and what grants.new says of each password. A scene
// not signed in reads only the grants a VM holds.
func (f Scene) signIns(full bool) *pagemenubar.SignIns {
	out := pagemenubar.SignIns{}
	if full {
		out.Vault, out.Offers = f.vault()
	}
	for _, h := range f.All.Macs {
		m := pagemenubar.MachineSignIns{Name: h.Name, Mac: h.Name, ThisMac: h.ThisMac}
		if !h.ThisMac && !h.Reachable {
			m.Silent = page.MacName(h.Name, proName) + " does not answer"
			out.Machines = append(out.Machines, m)
			continue
		}
		if full {
			m.Rows = f.programRows(h.Name, accounts.Where{})
		}
		out.Machines = append(out.Machines, m)
		for _, vm := range h.VMs {
			if vm.State != model.Running {
				continue
			}
			rows := f.programRows(vm.Name, accounts.Where{Kind: vm.Lifetime.Kind()})
			if !full {
				rows = slices.DeleteFunc(rows, func(r accounts.ProgramRow) bool { return r.How != accounts.HowGranted })
			}
			out.Machines = append(out.Machines, pagemenubar.MachineSignIns{Name: vm.Name, Mac: h.Name, ThisMac: h.ThisMac, VM: true, Rows: rows})
		}
	}
	return &out
}

func (f Scene) report(name string) heldMachine {
	for _, m := range f.Accounts.Machines {
		if m.Name == name {
			return m
		}
	}
	return heldMachine{}
}

func (f Scene) programRows(name string, where accounts.Where) []accounts.ProgramRow {
	var rows []accounts.ProgramRow
	for _, a := range f.report(name).Accounts {
		program, ok := programOf(a.Provider)
		if !ok {
			continue
		}
		rows = append(rows, accounts.Describe(factsOf(program, a), where))
	}
	return rows
}

func programOf(provider string) (accounts.Program, bool) {
	for _, p := range catalog.Providers {
		if p.ID != provider {
			continue
		}
		for _, known := range catalog.LoginPrograms {
			if known.Label == p.Label {
				return accounts.Program(known.Name), true
			}
		}
	}
	return "", false
}

func entryID(provider string) string { return "pw-" + provider }

func entryLabel(program accounts.Program) string { return program.Label() + " token" }

func factsOf(program accounts.Program, a heldAccount) accounts.Facts {
	facts := accounts.Facts{Program: program, State: accounts.ProgramSignedOut, How: accounts.HowOwn}
	if len(a.Identity) > 0 && a.Identity[0] != "?" {
		facts.Who = a.Identity[0]
	}
	switch {
	case a.Granted:
		facts.State, facts.How, facts.Entry, facts.EntryLabel = accounts.ProgramSignedIn, accounts.HowGranted, entryID(a.Provider), entryLabel(program)
	case a.State == catalog.StateOK:
		facts.State, facts.Scopes = accounts.ProgramSignedIn, a.Scopes
	case a.State == catalog.StateBroken:
		facts.State, facts.Reason = accounts.ProgramBroken, a.Detail
	}
	return facts
}

func (f Scene) vault() (vault *accounts.VaultRead, offers map[string]accounts.GrantOffer) {
	read := accounts.VaultRead{Mac: proName}
	offers = map[string]accounts.GrantOffer{}
	for _, a := range f.report(proName).Accounts {
		program, ok := programOf(a.Provider)
		if !ok || a.State != catalog.StateOK {
			continue
		}
		id := entryID(a.Provider)
		entry := accounts.VaultEntry{ID: id, Label: entryLabel(program), Program: program, Kind: accounts.KeyToken, Added: storyNow, Reach: accounts.ReachNotGranted}
		var candidates accounts.Candidates
		for _, h := range f.All.Macs {
			for _, vm := range h.VMs {
				if vm.State != model.Running {
					continue
				}
				c := accounts.Candidate{ToKind: accounts.ToMachine, To: vm.Name, Where: accounts.Where{Kind: vm.Lifetime.Kind()}}
				if !h.ThisMac {
					c.Where.FarMac = h.Name
				}
				for _, held := range f.report(vm.Name).Accounts {
					if held.Provider != a.Provider {
						continue
					}
					c.Facts = factsOf(program, held)
					c.Granted = held.Granted
				}
				if c.Granted {
					entry.Grants = append(entry.Grants, accounts.VaultGrant{ToKind: accounts.ToMachine, To: vm.Name})
					entry.Reach = accounts.ReachGranted
				}
				candidates.List = append(candidates.List, c)
			}
		}
		read.Entries = append(read.Entries, entry)
		offers[id] = accounts.Offer(candidates)
	}
	return &read, offers
}
