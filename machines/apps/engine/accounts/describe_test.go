package accounts_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func TestABrokenSignInSaysWhyWhereItCanBeSignedInAgain(t *testing.T) {
	broken := accounts.Facts{Program: accounts.Program("wrangler"), State: accounts.ProgramBroken, Reason: "Refused at the provider"}
	for name, c := range map[string]struct {
		facts  accounts.Facts
		where  accounts.Where
		action accounts.ProgramAction
		line   string
	}{
		"its reason":        {broken, accounts.Where{Kind: machine.Computer}, accounts.ActionSignInAgain, "Refused at the provider"},
		"a default":         {accounts.Facts{Program: broken.Program, State: accounts.ProgramBroken}, accounts.Where{Kind: machine.Computer}, accounts.ActionSignInAgain, "The sign-in no longer works"},
		"a stopped machine": {broken, accounts.Where{Kind: machine.Computer, Stopped: true}, accounts.ActionNone, "Start the machine first"},
	} {
		row := accounts.Describe(c.facts, c.where)
		if row.Action != c.action || row.Line != c.line {
			t.Errorf("%s: action %q, line %q; want %q, %q", name, row.Action, row.Line, c.action, c.line)
		}
	}
}

func TestAProgramSignedInOnlyByAGrantIsNeverOfferedASignIn(t *testing.T) {
	byGrant := aProgramSignedInOnlyByAGrant(t)
	for name, c := range map[string]struct {
		facts  accounts.Facts
		where  accounts.Where
		action accounts.ProgramAction
		line   string
	}{
		"signed out on a Mac":       {accounts.Facts{Program: byGrant, State: accounts.ProgramSignedOut}, accounts.Where{Kind: machine.Computer}, accounts.ActionNone, "Sign it in with its own login in a terminal"},
		"signed out in a workspace": {accounts.Facts{Program: byGrant, State: accounts.ProgramSignedOut}, accounts.Where{Kind: machine.Workspace}, accounts.ActionNone, "Grant it a password from its Mac"},
		"broken":                    {accounts.Facts{Program: byGrant, State: accounts.ProgramBroken, Reason: "Refused at the provider"}, accounts.Where{Kind: machine.Workspace}, accounts.ActionNone, "Refused at the provider"},
		"granted":                   {accounts.Facts{Program: byGrant, State: accounts.ProgramSignedIn, How: accounts.HowGranted}, accounts.Where{Kind: machine.Workspace}, accounts.ActionRemove, ""},
		"signed in by hand":         {accounts.Facts{Program: byGrant, State: accounts.ProgramSignedIn, How: accounts.HowOwn}, accounts.Where{Kind: machine.Computer}, accounts.ActionSignOut, ""},
	} {
		row := accounts.Describe(c.facts, c.where)
		if row.Action != c.action || row.Line != c.line {
			t.Errorf("%s: action %q, line %q; want %q, %q", name, row.Action, row.Line, c.action, c.line)
		}
	}
}

func aProgramSignedInOnlyByAGrant(t *testing.T) accounts.Program {
	t.Helper()
	for _, p := range accounts.Programs() {
		if !p.SignsInItself() {
			return p
		}
	}
	t.Fatal("the catalog signs in no program only by a grant")
	return ""
}
