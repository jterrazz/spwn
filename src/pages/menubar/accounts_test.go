package menubar

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/catalog"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

// invariant: SPEC A4 Accounts: the fleet's Signed in lists the account the
// most Macs hold first, each with the Macs that hold it, This Mac's first.
func TestTheAccountMostMacsHoldIsSignedInFirst(t *testing.T) {
	signed := func(provider string) accounts.Account {
		return accounts.Account{Provider: provider, State: catalog.StateOK, Origin: accounts.OriginOwn, Identity: []string{"jterrazz"}}
	}
	all := accounts.AllMachines{Now: macNow, Macs: []accounts.MacAccounts{
		{Name: proMac, ThisMac: true, Accounts: []accounts.Account{signed(catalog.ProviderDocker), signed(catalog.ProviderClaude)}},
		{Name: miniMac, Accounts: []accounts.Account{signed(catalog.ProviderClaude)}},
	}}
	p := Accounts(all, AccountsOf{}, page.Context{Machine: proMac})
	s := sectionOf(p, SectionSignedInAll)
	if s == nil {
		t.Fatal("no Signed in section")
	}
	got := make([]string, 0, len(s.Items))
	for _, it := range s.Items {
		got = append(got, it.Row.ID+": "+words(it.Row.Detail))
	}
	if want := []string{catalog.ProviderClaude + ": This Mac | mac-mini", catalog.ProviderDocker + ": This Mac"}; !slices.Equal(got, want) {
		t.Errorf("signed in %v, want %v", got, want)
	}
}

// invariant: SPEC A4 A workspace: a running workspace's page carries its
// header and four segments, its loans under Accounts alone; a stopped one
// carries no segment.
func TestAWorkspacesLoansAreItsAccountsSegment(t *testing.T) {
	report := &accounts.Report{Machines: []accounts.Machine{{Name: "linux-3", Kind: accounts.KindVM, Accounts: []accounts.Account{
		{Provider: catalog.ProviderGitHub, State: catalog.StateOK, Origin: accounts.OriginShared, Scopes: []string{"session:github", "read only"},
			Shared: &accounts.Loan{At: macNow.Add(-time.Hour), Until: macNow.Add(3 * time.Hour)}},
	}}}}
	p := macPageOf(t, "linux-3", func(m *Machine) { m.Accounts, m.Summary = report, nil })
	if p.Header == nil || p.Header.Back != "All machines" || len(p.Segments) != 4 {
		t.Fatalf("header %+v, segments %+v: want the way back to All machines and the four segments", p.Header, p.Segments)
	}
	for _, s := range p.Sections {
		switch {
		case s.ID == SectionShared && s.Segment != SegmentAccounts:
			t.Errorf("section %s is under %q, want %q", s.ID, s.Segment, SegmentAccounts)
		case s.ID == SectionSummary && s.Segment != "":
			t.Errorf("section %s is under %q, want none", s.ID, s.Segment)
		case s.ID != SectionShared && s.ID != SectionSummary && (s.Segment == "" || s.Segment == SegmentAccounts):
			t.Errorf("section %s is under %q, want a segment other than Accounts", s.ID, s.Segment)
		}
	}
	for _, a := range p.Actions {
		if (a.ID == ActionTakeBack+"/"+catalog.ProviderGitHub || a.ID == ActionExtend+miniMac+"/linux-3/"+catalog.ProviderGitHub) && (a.Run.Target == nil || a.Run.Target.Name != miniMac) {
			t.Errorf("%s runs on %+v, want on mac-mini, which lent it", a.ID, a.Run.Target)
		}
	}
	if s := sectionOf(p, SectionShared); s == nil || s.Title != "Lent to linux-3" || s.Items[0].Row == nil || words(s.Items[0].Row.Detail) != "read only" {
		t.Errorf("Accounts is %+v, want GitHub lent to linux-3, read only", s)
	}
	stopped := macPageOf(t, "linux-3", func(m *Machine) {
		m.Accounts, m.Summary = report, nil
		for i := range m.All.Macs[1].VMs {
			m.All.Macs[1].VMs[i].State = model.Stopped
		}
	})
	if len(stopped.Segments) != 0 {
		t.Errorf("a stopped workspace has segments %+v, want none", stopped.Segments)
	}
}

// invariant: Extend shares the copy again for 4 hr from now, as
// accounts.share records it, so its HUD says 4 hr left, never the time
// that was left plus 4 hr.
func TestExtendSaysTheCopyHas4HrLeftFromNow(t *testing.T) {
	report := &accounts.Report{Machines: []accounts.Machine{{Name: "linux-3", Kind: accounts.KindVM, Accounts: []accounts.Account{
		{Provider: catalog.ProviderGitHub, State: catalog.StateOK, Origin: accounts.OriginShared, Scopes: []string{"session:github"},
			Shared: &accounts.Loan{At: macNow.Add(-4 * time.Hour), Until: macNow.Add(24 * time.Minute)}},
	}}}}
	p := macPageOf(t, "linux-3", func(m *Machine) { m.Accounts, m.Summary = report, nil })
	var a *page.Action
	for i := range p.Actions {
		if p.Actions[i].ID == ActionExtend+miniMac+"/linux-3/"+catalog.ProviderGitHub {
			a = &p.Actions[i]
		}
	}
	if a == nil || a.Hud == nil || words(*a.Hud) != "GitHub on linux-3 | 4 hr left" || a.Run.In["For"] != strconv.FormatInt(int64(4*time.Hour), 10) {
		t.Errorf("Extend is %+v, want accounts.share for 4 hr and its HUD GitHub on linux-3, 4 hr left", a)
	}
}
