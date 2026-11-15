package accounts_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestABrokenSignInIsFixedBySigningInToThatProgramAgain(t *testing.T) {
	// invariant: only a broken sign-in is a finding, and its fix is the
	// accounts.login call of its own program.
	broken := accounts.ProgramOf(catalog.ProviderCloudflare)
	read := accounts.ProgramsRead{Programs: []accounts.ProgramRow{
		{Program: accounts.ProgramOf(catalog.ProviderGitHub), State: accounts.ProgramSignedIn},
		{Program: broken, State: accounts.ProgramBroken, Line: "Refused at the provider"},
		{Program: accounts.Program("other"), State: accounts.ProgramSignedOut},
	}}
	got := accounts.Findings(read, machine.Ref{Name: "linux-1"})
	if len(got) != 1 {
		t.Fatalf("%d findings, want the broken one alone", len(got))
	}
	if f := got[0].Fix; f.Call != "accounts.login" || len(f.Args) != 1 || f.Args["Program"] != string(broken) || f.On != "" || f.Command != "" {
		t.Errorf("fix %+v", f)
	}
}
