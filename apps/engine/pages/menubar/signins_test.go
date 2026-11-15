package menubar

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

func programFor(t *testing.T, provider string) accounts.Program {
	t.Helper()
	for _, p := range catalog.Providers {
		if p.ID != provider {
			continue
		}
		for _, known := range catalog.LoginPrograms {
			if known.Label == p.Label {
				return accounts.Program(known.Name)
			}
		}
	}
	t.Fatalf("no login program for %s", provider)
	return ""
}

func signedInRow(t *testing.T, provider, who string) accounts.ProgramRow {
	t.Helper()
	return accounts.Describe(accounts.Facts{Program: programFor(t, provider), State: accounts.ProgramSignedIn, How: accounts.HowOwn, Who: who}, accounts.Where{})
}

func signedOutRow(t *testing.T, provider string) accounts.ProgramRow {
	t.Helper()
	return accounts.Describe(accounts.Facts{Program: programFor(t, provider), State: accounts.ProgramSignedOut}, accounts.Where{})
}

func grantedRow(t *testing.T, provider, entry string) accounts.ProgramRow {
	t.Helper()
	return accounts.Describe(accounts.Facts{Program: programFor(t, provider), State: accounts.ProgramSignedIn, How: accounts.HowGranted, Entry: entry, EntryLabel: entry + " token"}, accounts.Where{})
}

func offerFor(vms ...string) accounts.GrantOffer {
	var offer accounts.GrantOffer
	for _, vm := range vms {
		offer.Targets = append(offer.Targets, accounts.Target{ToKind: accounts.ToMachine, To: vm, Fit: accounts.TargetFits, Line: "Signed out here"})
	}
	offer.Presence = "Touch ID will ask you to confirm before the password leaves this Mac"
	return offer
}
