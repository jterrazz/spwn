package pages

import (
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/catalog"
	pageaccounts "github.com/jterrazz/studio-station/src/pages/accounts"
)

var accountsNow = time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)

func own(provider string, identity, scopes []string, source, detail string) accounts.Account {
	return accounts.Account{Provider: provider, State: catalog.StateOK, Origin: accounts.OriginOwn, Identity: identity, Scopes: scopes, Source: source, Detail: detail}
}

func shared(provider, name string, left time.Duration, state catalog.State, reason, hint string) accounts.Account {
	return accounts.Account{
		Provider: provider, State: state, Reason: reason, Origin: accounts.OriginShared, Identity: []string{"admin@jterrazz.com"}, Scopes: []string{name}, Hint: hint,
		Shared: &accounts.Loan{Name: name, At: accountsNow.Add(left - 12*time.Hour), Until: accountsNow.Add(left)},
	}
}

func macAccounts() accounts.Machine {
	return accounts.Machine{Name: "jterrazz-macbook-pro", Kind: accounts.KindComputer, ThisMac: true, Accounts: []accounts.Account{
		github(),
		{Provider: catalog.ProviderNPM, State: catalog.StateBroken, Reason: accounts.ReasonRefused, Origin: accounts.OriginOwn, Identity: []string{"?"}, Detail: "token refused by registry.npmjs.org", Hint: "npm login"},
		own(catalog.ProviderClaude, []string{"admin@jterrazz.com"}, []string{"max"}, "", "claude.ai"),
		own(catalog.ProviderTailscale, []string{"jterrazz@github"}, []string{"jterrazz.github"}, "", "app-store"),
		{Provider: catalog.ProviderCloudflare, State: catalog.StateAbsent, Reason: accounts.ReasonMissing, Detail: "wrangler not on PATH"},
		own(catalog.ProviderGit, []string{"Jean-Baptiste Terrazzoni", "admin@jterrazz.com"}, nil, "", ""),
		own(catalog.ProviderSSH, []string{"6 keys"}, []string{"admin@jterrazz.com", "station-image", "station-e2e", "deploy", "old-rsa", "github-actions"}, "", ""),
		own(catalog.ProviderDocker, []string{"deploy"}, []string{"registry.jterrazz.com"}, "osxkeychain", ""),
	}}
}

func entryAccount() accounts.Account {
	return own(catalog.ProviderEntry, []string{"1 SSH key"}, []string{"jterrazz-macbook-pro"}, "", "SHA256:76MCcfWNZv0Ixcc4fM0b6Ue1o8l7W1Xo0n3zKhK1Vy4")
}

func accountsReport() accounts.Report {
	return accounts.Report{Now: accountsNow, Engine: accounts.EngineRunning, Machines: []accounts.Machine{
		macAccounts(),
		{Name: "linux-1", Kind: accounts.KindVM, Lifetime: "throwaway", Running: true, Guest: accounts.GuestConnected, Accounts: []accounts.Account{
			{Provider: catalog.ProviderGitHub, State: catalog.StateAbsent, Reason: accounts.ReasonUnset, Detail: "not logged in", Hint: "ssh linux-1 gh auth login"},
			{Provider: catalog.ProviderNPM, State: catalog.StateAbsent, Reason: accounts.ReasonMissing, Detail: "npm not on PATH"},
			shared(catalog.ProviderClaude, catalog.ClaudeTokenName, 11*time.Hour+40*time.Minute, catalog.StateOK, "", ""),
			own(catalog.ProviderGit, []string{"Jean-Baptiste Terrazzoni", "admin@jterrazz.com"}, nil, "", ""),
			{Provider: catalog.ProviderSSH, State: catalog.StateAbsent, Reason: accounts.ReasonUnset, Detail: "no key in the agent"},
			entryAccount(),
		}},
		{Name: "api", Kind: accounts.KindVM, Lifetime: "permanent", Running: true, Guest: accounts.GuestConnected, Accounts: []accounts.Account{
			shared(catalog.ProviderClaude, catalog.ClaudeTokenName, 42*time.Minute, catalog.StateDegraded, accounts.ReasonExpiring, "station accounts share claude --with api"),
			entryAccount(),
		}},
	}}
}

func emptyAccounts() accounts.Report {
	return accounts.Report{Now: accountsNow, Engine: accounts.EngineRunning, Machines: []accounts.Machine{{Name: "jterrazz-macbook-pro", Kind: accounts.KindComputer, ThisMac: true, Accounts: []accounts.Account{
		{Provider: catalog.ProviderGitHub, State: catalog.StateAbsent, Reason: accounts.ReasonUnset, Detail: "not logged in", Hint: "station settings apply github-login"},
		{Provider: catalog.ProviderNPM, State: catalog.StateAbsent, Reason: accounts.ReasonUnset, Detail: "not logged in", Hint: "npm login"},
		{Provider: catalog.ProviderClaude, State: catalog.StateAbsent, Reason: accounts.ReasonUnset, Detail: "not logged in", Hint: "claude /login"},
		{Provider: catalog.ProviderTailscale, State: catalog.StateAbsent, Reason: accounts.ReasonMissing, Detail: "not installed"},
		{Provider: catalog.ProviderGit, State: catalog.StateAbsent, Reason: accounts.ReasonUnset, Detail: "no user.name nor user.email", Hint: "git config --global user.name <name>"},
	}}}}
}

func stoppedAccounts() accounts.Report {
	r := accountsReport()
	r.Engine = accounts.EngineStopped
	return r
}

func loadingAccounts() accounts.Report {
	r := accountsReport()
	for i := range r.Machines[0].Accounts {
		r.Machines[0].Accounts[i] = accounts.Account{Provider: r.Machines[0].Accounts[i].Provider, State: catalog.StatePending}
	}
	r.Machines[1].Pending, r.Machines[1].Accounts = true, r.Machines[1].Accounts[2:3]
	r.Machines[2].Pending, r.Machines[2].Accounts = true, r.Machines[2].Accounts[:1]
	return r
}

func accountPages() []Fixture {
	return states("accounts",
		pageaccounts.Page(loadingAccounts(), thisMacContext()),
		pageaccounts.Page(emptyAccounts(), thisMacContext()),
		pageaccounts.Page(stoppedAccounts(), thisMacContext()),
		pageaccounts.Page(accountsReport(), thisMacContext()),
	)
}

func github() accounts.Account {
	a := own(catalog.ProviderGitHub, []string{"jterrazz"}, []string{"gist", "read:org", "repo", "workflow"}, "gh", "")
	a.Notes = []string{"keyring", "ssh"}
	return a
}
