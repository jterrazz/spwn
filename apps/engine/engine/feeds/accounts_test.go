package feeds_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/feeds"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

const air = "macbook-air"

func signedIn(provider, who string, scopes ...string) accounts.ProgramRow {
	return accounts.Describe(accounts.Facts{Program: programNamed(provider), State: accounts.ProgramSignedIn, How: accounts.HowOwn, Who: who, Scopes: scopes}, accounts.Where{})
}

func twoMacsSignedIn() *world {
	linux := machines.VM{Name: "linux-1", Mac: thisMac, Lifetime: machines.Persistent, State: machines.Running, Guest: "connected"}
	refused := accounts.Describe(accounts.Facts{Program: programNamed(catalog.ProviderCloudflare), State: accounts.ProgramBroken, Reason: "Its token was refused"}, accounts.Where{})
	entry := func(provider string) accounts.VaultEntry {
		return accounts.VaultEntry{ID: "pw-" + provider, Label: provider + " token", Program: programNamed(provider)}
	}
	offer := accounts.Offer([]accounts.Candidate{{To: "linux-1"}})
	return &world{infra: []machines.VM{infraAt(machines.Running, 0)}, here: []machines.VM{linux}, asked: map[string][]string{}, activity: map[string]uint64{},
		added: []machines.MacView{{Name: air}},
		programs: map[string][]accounts.ProgramRow{
			thisMac + "/":        {signedIn(catalog.ProviderGitHub, "you", "repo", "workflow"), refused, signedIn(catalog.ProviderClaude, "you@example.com")},
			thisMac + "/linux-1": grantedTo(catalog.ProviderGitHub, catalog.ProviderClaude),
			mini + "/":           {signedIn(catalog.ProviderGitHub, "ci-bot", "repo"), signedIn(catalog.ProviderClaude, "")},
		},
		vault:  accounts.VaultRead{Mac: thisMac, Entries: []accounts.VaultEntry{entry(catalog.ProviderGitHub), entry(catalog.ProviderClaude)}},
		offers: map[string]accounts.GrantOffer{"pw-" + catalog.ProviderGitHub: offer, "pw-" + catalog.ProviderClaude: offer}}
}

func watchAccounts(t *testing.T, w *world, subject string) page.Page {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	pages := feeds.Menubar{Calls: w, Clock: newSteps(), ThisMachine: page.Context{Machine: thisMac, Kind: "computer"}}.Accounts()(ctx, engine.FeedInput{Subject: subject})
	return until(t, pages, "the accounts read", func(p page.Page) bool { return p.State != page.Loading })
}

func sectionIDs(p page.Page) []string {
	out := make([]string, 0, len(p.Sections))
	for _, s := range p.Sections {
		out = append(out, s.ID)
	}
	return out
}

func rowIDs(p page.Page, section string) []string {
	var out []string
	for _, s := range p.Sections {
		if s.ID != section {
			continue
		}
		for _, it := range s.Items {
			if it.Row != nil {
				out = append(out, it.Row.ID)
			}
		}
	}
	return out
}

func TestEveryMacsSignInsAreReadThroughItsOwnEngineWhatNeedsYouFirst(t *testing.T) {
	w := twoMacsSignedIn()
	p := watchAccounts(t, w, "")
	w.mu.Lock()
	read := slices.Clone(w.read)
	w.mu.Unlock()
	if want := []string{thisMac + "/", thisMac + "/linux-1", mini + "/", mini + "/infra"}; !slices.Equal(read, want) {
		t.Errorf("sign-ins read on %v, want each Mac then its running VMs, the unreachable air not asked", read)
	}
	if got, want := sectionIDs(p), []string{"summary", "needs-you", "on:" + thisMac, "on:linux-1", "on:" + mini, "silent"}; !slices.Equal(got, want) {
		t.Fatalf("sections %v, want %v", got, want)
	}
	if got, want := rowIDs(p, "needs-you"), []string{thisMac + "/" + string(programNamed(catalog.ProviderCloudflare))}; !slices.Equal(got, want) {
		t.Errorf("needs you %v, want Cloudflare signed out", got)
	}
	if got := words(p.Sections[0].Caption); got != "All machines | 4 sign-ins | 2 grants" {
		t.Errorf("the page's facts are %q, want where, the sign-ins and the grants", got)
	}
	if r := row(p, "silent", air); r == nil || r.Cells[0].Text != "macbook-air does not answer" {
		t.Errorf("the air is %+v, want it named as not answering", r)
	}
}

func TestASignInSaysWhatItsProviderPrintedAndAGrantItsPassword(t *testing.T) {
	p := watchAccounts(t, twoMacsSignedIn(), "")
	github, claude := string(programNamed(catalog.ProviderGitHub)), string(programNamed(catalog.ProviderClaude))
	for _, c := range []struct {
		section, id string
		want        []string
	}{
		{"on:" + thisMac, thisMac + "/" + github, []string{"you", "repo, workflow"}},
		{"on:" + thisMac, thisMac + "/" + claude, []string{"you@example.com"}},
		{"on:" + mini, mini + "/" + github, []string{"ci-bot", "repo"}},
		{"on:linux-1", "linux-1/" + github, []string{"from the password " + catalog.ProviderGitHub + " token"}},
	} {
		r := row(p, c.section, c.id)
		if r == nil {
			t.Errorf("no row %s", c.id)
			continue
		}
		if got := texts(r.Detail); !slices.Equal(got, c.want) {
			t.Errorf("%s says %q, want %q", c.id, got, c.want)
		}
	}
	if out := row(p, "needs-you", thisMac+"/"+string(programNamed(catalog.ProviderCloudflare))); out == nil || out.Mark != page.Broken || out.Reason != page.Refused {
		t.Errorf("Cloudflare needs you as %+v, want broken and refused", out)
	}
}

func texts(v page.Value) []string {
	if v.Kind != page.KindList {
		if v.Text == "" {
			return nil
		}
		return []string{v.Text}
	}
	out := make([]string, 0, len(v.Items))
	for _, it := range v.Items {
		out = append(out, it.Text)
	}
	return out
}

func TestOneMacsSignInsAreThatMacsAloneReadThroughItsEngine(t *testing.T) {
	w := twoMacsSignedIn()
	p := watchAccounts(t, w, "mac-mini")
	w.mu.Lock()
	read := slices.Clone(w.read)
	w.mu.Unlock()
	if want := []string{mini + "/", mini + "/infra"}; !slices.Equal(read, want) {
		t.Errorf("sign-ins read on %v, want the mini and its VM alone", read)
	}
	if got, want := sectionIDs(p), []string{"summary", "on:" + mini}; !slices.Equal(got, want) {
		t.Errorf("the mini's accounts are %v, want %v", got, want)
	}
}

func TestAWorkspacesAccountsAreTheGrantsMadeToIt(t *testing.T) {
	p := watchAccounts(t, twoMacsSignedIn(), "linux-1")
	github, claude := string(programNamed(catalog.ProviderGitHub)), string(programNamed(catalog.ProviderClaude))
	if got, want := sectionIDs(p), []string{"summary", "on:linux-1"}; !slices.Equal(got, want) {
		t.Fatalf("linux-1's accounts are %v, want %v", got, want)
	}
	if got, want := rowIDs(p, "on:linux-1"), []string{"linux-1/" + github, "linux-1/" + claude, "grant-password"}; !slices.Equal(got, want) {
		t.Errorf("linux-1's grants are %v, want its two, then Grant a password", got)
	}
	w := twoMacsSignedIn()
	w.beside = []machines.VM{{Name: "review-bot", Mac: mini, Lifetime: machines.Ephemeral, State: machines.Running}}
	bot := watchAccounts(t, w, "review-bot")
	if s := bot.Sections; len(s) != 2 || s[1].Items[0].Empty == nil || s[1].Items[0].Empty.Noun != "accounts" {
		t.Errorf("review-bot's accounts are %+v, want none, said so", s)
	}
}

// invariant: Home reads every machine's sign-ins: its Accounts tile counts
// the grants made to VMs.
func TestTheAccountsTileCountsTheGrantsOfEveryMachine(t *testing.T) {
	w := twoMacsSignedIn()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	o := feeds.Menubar{Calls: w, Clock: newSteps(), ThisMachine: page.Context{Machine: thisMac, Kind: "computer"}, Report: func(awake.View) *awake.Report { return nil }}
	in := engine.FeedInput{}
	in.Awake = nothing(in.Awake)
	until(t, o.Feed()(ctx, in), "the Accounts tile", func(p page.Page) bool {
		for _, s := range p.Sections {
			for _, it := range s.Items {
				if it.Tile != nil && it.Tile.Caption == "Accounts" && it.Tile.Number.N == 2 && it.Tile.Word == "granted" {
					return true
				}
			}
		}
		return false
	})
}

func words(vs []page.Value) string {
	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, v.Text)
	}
	return strings.Join(out, " | ")
}

func action(p page.Page, id string) *page.Action {
	for i := range p.Actions {
		if p.Actions[i].ID == id {
			return &p.Actions[i]
		}
	}
	return nil
}

func TestASignInOpensTerminalOnTheLoginAndWaitsWhileItRuns(t *testing.T) {
	p := watchAccounts(t, twoMacsSignedIn(), "")
	r := row(p, "needs-you", thisMac+"/"+string(programNamed(catalog.ProviderCloudflare)))
	if r.Label != "This Mac" || !slices.Equal(texts(r.Detail), []string{"Cloudflare needs sign-in", "Its token was refused"}) {
		t.Errorf("the sign-in reads %q, %q, want This Mac, Cloudflare needs sign-in, why", r.Label, texts(r.Detail))
	}
	a := action(p, r.Verb)
	if a == nil || a.Label != "Sign in" || a.Busy != "Waiting" || a.Run == nil || !a.Run.Terminal || !slices.Equal(a.Run.Argv, []string{"accounts"}) {
		t.Fatalf("Sign in is %+v, want a terminal on station accounts, busy Waiting while it runs", a)
	}
	if a.Hud == nil || len(a.Hud.Items) != 2 || a.Hud.Items[1].Text != "station accounts on This Mac" {
		t.Errorf("Sign in's HUD is %+v, want Terminal and the command it runs, on This Mac", a.Hud)
	}
}

func TestTheFleetPageGrantsFromItsFooterAndEachGrantOffersRemove(t *testing.T) {
	p := watchAccounts(t, twoMacsSignedIn(), "")
	if p.Footer == nil || !slices.Equal(p.Footer.Buttons, []string{"full-list", "grant"}) {
		t.Fatalf("the footer is %+v, want Full list in Terminal then Grant a password", p.Footer)
	}
	if full := action(p, "full-list"); full == nil || !slices.Equal(full.Run.Argv, []string{"accounts", "--all-machines"}) {
		t.Errorf("Full list in Terminal is %+v, want station accounts --all-machines", full)
	}
	if slices.Contains(rowIDs(p, "on:linux-1"), "grant-password") {
		t.Error("the fleet page grants from its footer, never from a row")
	}
	remove := action(p, "remove-grant:linux-1/"+string(programNamed(catalog.ProviderGitHub)))
	if remove == nil || remove.Run == nil || remove.Run.Call != "grants.remove" || remove.Run.In["Entry"] != "pw-"+catalog.ProviderGitHub || remove.Run.In["To"] != "linux-1" {
		t.Errorf("Remove is %+v, want grants.remove of GitHub's password from linux-1", remove)
	}
}
