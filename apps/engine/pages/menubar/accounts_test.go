package menubar

import (
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func signInsOfTwo(t *testing.T) SignIns {
	t.Helper()
	return SignIns{Machines: []MachineSignIns{
		{Name: proMac, Mac: proMac, ThisMac: true, Rows: []accounts.ProgramRow{signedInRow(t, catalog.ProviderGitHub, "you"), signedOutRow(t, catalog.ProviderClaude)}},
		{Name: "linux-1", Mac: proMac, ThisMac: true, VM: true, Rows: []accounts.ProgramRow{grantedRow(t, catalog.ProviderGitHub, "pw-1")}},
		{Name: miniMac, Mac: miniMac, Silent: "mac-mini does not answer"},
	}}
}

// invariant: SPEC A4 Accounts: its facts count the sign-ins and the
// grants; what needs you first, each sign-in a machine is out of with its
// Sign in; then each machine's card, a Mac that did not answer apart.
func TestTheAccountsPageSaysEachMachinesSignInsAndGrants(t *testing.T) {
	p := Accounts(signInsOfTwo(t), AccountsOf{Asks: yesNo}, page.Context{Machine: proMac})
	if s := sectionOf(p, SectionSummary); s == nil || len(s.Caption) != 3 || s.Caption[1].Text != "1 sign-in" || s.Caption[2].Text != "1 grant" {
		t.Fatalf("facts %+v, want where, 1 sign-in, 1 grant", s)
	}
	needs := sectionOf(p, SectionNeedsYou)
	if needs == nil || len(needs.Items) != 1 || needs.Items[0].Row.Detail.Text != "Claude needs sign-in" {
		t.Fatalf("needs %+v, want Claude's sign-in", needs)
	}
	grants := sectionOf(p, SectionOn+"linux-1")
	if grants == nil || grants.Title != "On linux-1" || len(grants.Items) != 1 || grants.Items[0].Row.Cells[0].Text != "granted" {
		t.Fatalf("linux-1's card %+v, want one granted row", grants)
	}
	if silent := sectionOf(p, SectionSilent); silent == nil || len(silent.Items) != 1 {
		t.Fatalf("not read %+v, want mac-mini", silent)
	}
}

// invariant: SPEC A5 Grant a password: the page's To popup swaps between
// the sheets of the machines it offers, so each machine it names has its
// sheet on the page.
func TestEachMachineTheGrantSheetOffersHasItsSheetOnTheAccountsPage(t *testing.T) {
	s := signInsOfTwo(t)
	s.Vault = &accounts.VaultRead{Mac: proMac, Entries: []accounts.VaultEntry{{ID: "pw-1", Label: "deploy token", Program: accounts.Programs()[0]}}}
	s.Offers = map[string]accounts.GrantOffer{"pw-1": offerFor("linux-1", "agent-7f")}
	p := Accounts(s, AccountsOf{Asks: yesNo}, page.Context{Machine: proMac})
	to := actionOf(p, ActionGrant).Form.Fields[0]
	if len(to.Options) != 2 {
		t.Fatalf("To offers %+v, want linux-1 and agent-7f", to.Options)
	}
	for _, o := range to.Options {
		if sheet := actionOf(p, o.Value); sheet.Sheet != page.SheetGrant || sheet.Run == nil || sheet.Run.In["To"] != o.Label {
			t.Errorf("To offers %s, whose action is %+v: want the grant sheet to %s", o.Value, sheet, o.Label)
		}
	}
}

// invariant: SPEC A4 A workspace: a running workspace's page carries its
// header and four segments, its grants under Accounts alone; a stopped one
// carries no segment.
func TestAWorkspacesGrantsAreItsAccountsSegment(t *testing.T) {
	s := SignIns{Machines: []MachineSignIns{{Name: "linux-3", Mac: miniMac, VM: true, Rows: []accounts.ProgramRow{grantedRow(t, catalog.ProviderGitHub, "pw-1")}}}}
	p := macPageOf(t, "linux-3", func(m *Machine) { m.SignIns, m.Summary = &s, nil })
	if p.Header == nil || p.Header.Back != "Back" || len(p.Segments) != 4 {
		t.Fatalf("header %+v, segments %+v: want the way Back and the four segments", p.Header, p.Segments)
	}
	for _, sec := range p.Sections {
		switch {
		case sec.ID == SectionShared && sec.Segment != SegmentAccounts:
			t.Errorf("section %s is under %q, want %q", sec.ID, sec.Segment, SegmentAccounts)
		case sec.ID == SectionSummary && sec.Segment != "":
			t.Errorf("section %s is under %q, want none", sec.ID, sec.Segment)
		case sec.ID != SectionShared && sec.ID != SectionSummary && (sec.Segment == "" || sec.Segment == SegmentAccounts):
			t.Errorf("section %s is under %q, want a segment other than Accounts", sec.ID, sec.Segment)
		}
	}
	acc := sectionOf(p, SectionShared)
	if acc == nil || acc.Title != "Accounts" || acc.Items[0].Row == nil || words(acc.Items[0].Row.Detail) != "from the password pw-1 token" {
		t.Errorf("Accounts is %+v, want GitHub granted from its password", acc)
	}
	stopped := macPageOf(t, "linux-3", func(m *Machine) {
		m.SignIns, m.Summary = &s, nil
		for i := range m.All.Macs[1].VMs {
			m.All.Macs[1].VMs[i].State = model.Stopped
		}
	})
	if len(stopped.Segments) != 0 {
		t.Errorf("a stopped workspace has segments %+v, want none", stopped.Segments)
	}
}

// invariant: Remove ends a grant to a VM of This Mac through grants.remove,
// asking in the engine's words; a grant to a VM of another Mac is
// removed from that Mac, so its page offers none.
func TestRemovingAGrantAsksAndCallsGrantsRemoveOnThisMacAlone(t *testing.T) {
	p := Accounts(signInsOfTwo(t), AccountsOf{Asks: yesNo}, page.Context{Machine: proMac})
	var remove *page.Action
	for i := range p.Actions {
		if strings.HasPrefix(p.Actions[i].ID, ActionRemoveGrant) {
			remove = &p.Actions[i]
		}
	}
	if remove == nil || remove.Run.Call != CallRemoveGrant || remove.Run.In["Entry"] != "pw-1" || remove.Run.In["To"] != "linux-1" || remove.Run.In["Confirm"] != yes || remove.Confirm == nil {
		t.Fatalf("Remove is %+v, want grants.remove of pw-1 from linux-1, confirmed", remove)
	}
	if !remove.Destructive || !remove.Confirm.Danger {
		t.Errorf("Remove is red %v, its question red %v; want both, as the window draws it", remove.Destructive, remove.Confirm.Danger)
	}
	if remove.Hud == nil || words(*remove.Hud) != "GitHub removed from linux-1" {
		t.Errorf("Remove says %+v once it ran", remove.Hud)
	}
	far := SignIns{Machines: []MachineSignIns{{Name: "linux-3", Mac: miniMac, VM: true, Rows: []accounts.ProgramRow{grantedRow(t, catalog.ProviderGitHub, "pw-1")}}}}
	if got := Accounts(far, AccountsOf{Asks: yesNo}, page.Context{Machine: proMac}); slices.ContainsFunc(got.Actions, func(a page.Action) bool { return strings.HasPrefix(a.ID, ActionRemoveGrant) }) {
		t.Errorf("a grant to a VM of another Mac offers Remove")
	}
}
