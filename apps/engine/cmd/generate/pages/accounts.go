package pages

import (
	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	pageaccounts "github.com/jterrazz/studio-station/apps/engine/pages/accounts"
)

type heldAccount struct {
	Provider string
	State    catalog.State
	Identity []string
	Scopes   []string
	Detail   string
	Granted  bool
}

type heldMachine struct {
	Name     string
	Accounts []heldAccount
}

type heldReport struct{ Machines []heldMachine }

func own(provider string, identity, scopes []string, detail string) heldAccount {
	return heldAccount{Provider: provider, State: catalog.StateOK, Identity: identity, Scopes: scopes, Detail: detail}
}

func listed(rows ...accounts.Facts) pageaccounts.Machine {
	m := pageaccounts.Machine{Name: "macbook-pro", Kind: string(coremachine.Computer)}
	for _, f := range rows {
		m.Read.Programs = append(m.Read.Programs, accounts.Describe(f, accounts.Where{Kind: coremachine.Computer}))
	}
	return m
}

func everyProgram(state accounts.ProgramState) pageaccounts.Machine {
	facts := make([]accounts.Facts, 0, len(accounts.Programs()))
	for _, p := range accounts.Programs() {
		facts = append(facts, accounts.Facts{Program: p, State: state, How: accounts.HowOwn})
	}
	return listed(facts...)
}

func fullAccounts() pageaccounts.Machine {
	return listed(
		accounts.Facts{Program: accounts.ProgramOf(catalog.ProviderGitHub), State: accounts.ProgramSignedIn, Who: "you", Scopes: []string{"gist", "read:org", "repo", "workflow"}, How: accounts.HowOwn},
		accounts.Facts{Program: accounts.ProgramOf(catalog.ProviderNPM), State: accounts.ProgramBroken, Who: "you", How: accounts.HowOwn, Reason: "The token is refused by registry.npmjs.org"},
		accounts.Facts{Program: accounts.ProgramOf(catalog.ProviderClaude), State: accounts.ProgramSignedIn, Who: "you@example.com", Plan: "max", How: accounts.HowOwn},
		accounts.Facts{Program: accounts.ProgramOf(catalog.ProviderCloudflare), State: accounts.ProgramSignedOut, How: accounts.HowOwn},
	)
}

func accountPages() []Fixture {
	return states("accounts",
		pageaccounts.Page(everyProgram(accounts.ProgramReading), thisMacContext()),
		pageaccounts.Page(everyProgram(accounts.ProgramSignedOut), thisMacContext()),
		pageaccounts.Page(listed(accounts.Facts{Program: accounts.ProgramOf(catalog.ProviderGitHub), State: accounts.ProgramBroken, How: accounts.HowOwn}), thisMacContext()),
		pageaccounts.Page(fullAccounts(), thisMacContext()),
	)
}
