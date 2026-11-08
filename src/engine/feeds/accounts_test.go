package feeds_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/engine/feeds"
	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

const air = "jterrazz-macbook-air"

func twoMacsSignedIn() *world {
	now := newSteps().Now()
	linux := machines.VM{Name: "linux-1", Mac: thisMac, Lifetime: machines.Permanent, State: machines.Running, Guest: "connected"}
	return &world{infra: []machines.VM{infraAt(machines.Running, 0)}, here: []machines.VM{linux}, asked: map[string][]string{}, activity: map[string]uint64{},
		added: []machines.MacView{{Name: air}},
		signedIn: map[string]accounts.Report{
			thisMac: {Machines: []accounts.Machine{{Name: thisMac, Kind: accounts.KindMac, ThisMac: true, Accounts: []accounts.Account{
				{Provider: catalog.ProviderGitHub, State: catalog.StateOK, Identity: []string{"jterrazz"}, Scopes: []string{"repo", "workflow"}},
				{Provider: catalog.ProviderCloudflare, State: catalog.StateBroken, Reason: accounts.ReasonRefused, Identity: []string{accounts.Unknown}, Hint: "station settings apply cloudflare-login"},
				{Provider: catalog.ProviderClaude, State: catalog.StateOK, Identity: []string{"admin@jterrazz.com"}, Scopes: []string{"max"}},
				{Provider: catalog.ProviderGit, State: catalog.StateOK, Identity: []string{"Jean-Baptiste Terrazzoni"}},
			}}}, Shares: []accounts.Shared{
				{Account: catalog.ProviderGitHub, VM: "linux-1", At: now.Add(-8 * time.Hour), Until: now.Add(4 * time.Hour)},
				{Account: catalog.ProviderClaude, VM: "linux-1", At: now.Add(-3*time.Hour - 38*time.Minute), Until: now.Add(22 * time.Minute)},
			}},
			mini: {Machines: []accounts.Machine{{Name: mini, Kind: accounts.KindMac, Accounts: []accounts.Account{
				{Provider: catalog.ProviderGitHub, State: catalog.StateOK, Identity: []string{"jterrazz-bot"}, Scopes: []string{"repo"}},
				{Provider: catalog.ProviderClaude, State: catalog.StateOK, Scopes: []string{"max"}},
			}}}},
		}}
}

func watchAccounts(t *testing.T, w *world, subject string) page.Page {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	pages := feeds.Menubar{Calls: w, Clock: newSteps(), ThisMachine: page.Context{Machine: thisMac, Kind: "mac"}}.Accounts()(ctx, engine.FeedInput{Subject: subject})
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

func TestEveryMacsAccountsAreReadThroughItsOwnEngineWhatNeedsYouFirst(t *testing.T) {
	w := twoMacsSignedIn()
	p := watchAccounts(t, w, "")
	if got := w.askedOf(accountscalls.List.Name); !slices.Equal(got, []string{thisMac, mini}) {
		t.Errorf("accounts read on %v, want This Mac then the mini, the asleep air not asked", got)
	}
	if got, want := sectionIDs(p), []string{"summary", "needs-you", "shared", "signed-in", "silent"}; !slices.Equal(got, want) {
		t.Fatalf("sections %v, want %v", got, want)
	}
	if got, want := rowIDs(p, "needs-you"), []string{"short/" + id(thisMac, "linux-1", catalog.ProviderClaude), id(thisMac, catalog.ProviderCloudflare)}; !slices.Equal(got, want) {
		t.Errorf("needs you %v, want Claude's copy under 30 min, then Cloudflare signed out", got)
	}
	if got, want := rowIDs(p, "signed-in"), []string{catalog.ProviderGitHub, catalog.ProviderClaude}; !slices.Equal(got, want) {
		t.Errorf("signed in to %v, want GitHub and Claude once each, git being no sign-in", got)
	}
	if r := row(p, "signed-in", catalog.ProviderGitHub); r == nil || words(r.Detail.Items) != "This Mac | mac-mini" {
		t.Errorf("GitHub is signed in on %+v, want This Mac then mac-mini", r)
	}
	if got := words(p.Sections[0].Caption); got != "All machines | 2 accounts | 2 lent to VMs" {
		t.Errorf("the page's facts are %q, want where, the accounts signed in and the copies shared", got)
	}
	if r := row(p, "silent", air); r == nil || r.Cells[0].Text != "macbook-air does not answer" {
		t.Errorf("the air is %+v, want it named as not answering", r)
	}
}

func TestASignedInAccountSaysWhatItsProviderPrintedAndNothingMore(t *testing.T) {
	p, pro, minis := watchAccounts(t, twoMacsSignedIn(), ""), watchAccounts(t, twoMacsSignedIn(), thisMac), watchAccounts(t, twoMacsSignedIn(), mini)
	of := map[string]page.Page{thisMac: pro, mini: minis}
	if got, want := rowIDs(pro, "signed-in:"+thisMac), []string{id(thisMac, catalog.ProviderGitHub), id(thisMac, catalog.ProviderClaude)}; !slices.Equal(got, want) {
		t.Errorf("This Mac signed in to %v, want GitHub and Claude, git being no sign-in", got)
	}
	if s := minis.Sections[len(minis.Sections)-1]; s.Title != "Signed in on mac-mini" {
		t.Errorf("the mini's sign-ins are under %q, want the Mac by its name", s.Title)
	}
	for _, c := range []struct {
		section, id string
		want        []string
	}{
		{"signed-in:" + thisMac, id(thisMac, catalog.ProviderGitHub), []string{"jterrazz", "repo, workflow"}},
		{"signed-in:" + thisMac, id(thisMac, catalog.ProviderClaude), []string{"Max plan"}},
		{"signed-in:" + mini, id(mini, catalog.ProviderGitHub), []string{"jterrazz-bot", "repo"}},
		{"signed-in:" + mini, id(mini, catalog.ProviderClaude), []string{"Max plan"}},
	} {
		r := row(of[strings.TrimPrefix(c.section, "signed-in:")], c.section, c.id)
		if r == nil {
			t.Errorf("no row %s", c.id)
			continue
		}
		words := make([]string, 0, len(r.Detail.Items))
		for _, v := range r.Detail.Items {
			words = append(words, v.Text)
		}
		if !slices.Equal(words, c.want) {
			t.Errorf("%s says %q, want %q", c.id, words, c.want)
		}
	}
	if out := row(p, "needs-you", id(thisMac, catalog.ProviderCloudflare)); out.Mark != page.Broken || out.Reason != page.Refused || out.Hint == nil || out.Hint.Command != "station settings apply cloudflare-login" {
		t.Errorf("Cloudflare needs you as %+v, want broken, refused, with its fix", out)
	}
}

func id(parts ...string) string { return strings.Join(parts, "/") }

func TestEverySharedCopyCarriesItsTimeLeftAndTotalTheOneUnder30MinOrange(t *testing.T) {
	p := watchAccounts(t, twoMacsSignedIn(), "")
	if got, want := rowIDs(p, "shared"), []string{id(thisMac, "linux-1", catalog.ProviderClaude), id(thisMac, "linux-1", catalog.ProviderGitHub)}; !slices.Equal(got, want) {
		t.Fatalf("shared %v, want Claude ending first, then GitHub", got)
	}
	claude, github := row(p, "shared", id(thisMac, "linux-1", catalog.ProviderClaude)), row(p, "shared", id(thisMac, "linux-1", catalog.ProviderGitHub))
	if l := claude.Left; l == nil || l.N != 22*60 || l.Of != 4*3600 || claude.Mark != page.Attention || claude.Reason != page.Expiring {
		t.Errorf("Claude's copy is %+v left %+v, want 22 min of 4 hr, orange", claude, claude.Left)
	}
	if l := github.Left; l == nil || l.N != 4*3600 || l.Of != 12*3600 || github.Mark != page.OK {
		t.Errorf("GitHub's copy is %+v left %+v, want 4 hr of 12 hr, plain", github, github.Left)
	}
}

func TestOneMacsAccountsAreThatMacsAloneReadThroughItsEngine(t *testing.T) {
	w := twoMacsSignedIn()
	p := watchAccounts(t, w, "mac-mini")
	if got := w.askedOf(accountscalls.List.Name); !slices.Equal(got, []string{mini}) {
		t.Errorf("accounts read on %v, want the mini alone", got)
	}
	if got, want := sectionIDs(p), []string{"summary", "shared", "signed-in:" + mini}; !slices.Equal(got, want) {
		t.Errorf("the mini's accounts are %v, want %v", got, want)
	}
}

func TestAWorkspacesAccountsAreTheCopiesItsMacSharedWithIt(t *testing.T) {
	p := watchAccounts(t, twoMacsSignedIn(), "linux-1")
	if got, want := sectionIDs(p), []string{"summary", "needs-you", "shared"}; !slices.Equal(got, want) {
		t.Fatalf("linux-1's accounts are %v, want %v", got, want)
	}
	if s := p.Sections[2]; s.Title != "Lent to linux-1" || !slices.Equal(rowIDs(p, "shared"), []string{id(thisMac, "linux-1", catalog.ProviderClaude), id(thisMac, "linux-1", catalog.ProviderGitHub), "share-account"}) {
		t.Errorf("linux-1's copies are %q with %v, want its two, then Lend an account", s.Title, rowIDs(p, "shared"))
	}
	w := twoMacsSignedIn()
	w.beside = []machines.VM{{Name: "review-bot", Mac: mini, Lifetime: machines.Throwaway, State: machines.Running}}
	bot := watchAccounts(t, w, "review-bot")
	if s := bot.Sections; len(s) != 2 || s[1].Title != "Lent to review-bot" || s[1].Items[0].Empty == nil || s[1].Items[0].Empty.Text != "No account lent to review-bot" {
		t.Errorf("review-bot's accounts are %+v, want none shared, said so", s)
	}
}

func TestTheStatusItemCountsACopySharedForLessThan30MinMore(t *testing.T) {
	w := twoMacsSignedIn()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	o := feeds.Menubar{Calls: w, Clock: newSteps(), ThisMachine: page.Context{Machine: thisMac, Kind: "mac"}, Report: func(awake.View) *awake.Report { return nil }}
	in := engine.FeedInput{}
	in.Awake = nothing(in.Awake)
	p := until(t, o.Feed()(ctx, in), "the badge", func(p page.Page) bool { return p.Badge != nil })
	if want := (page.Badge{Needs: 1, Label: "station, 1 thing needs you"}); *p.Badge != want {
		t.Errorf("badge %+v, want %+v: Claude shared with linux-1 ends in 22 min", *p.Badge, want)
	}
}

// invariant: Home is read at the engine's clock: the hero counts down a
// copy's last minutes from now, not from when the machines answered.
func TestHomeSaysACopysLastMinutesAtTheEnginesClock(t *testing.T) {
	w := twoMacsSignedIn()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	o := feeds.Menubar{Calls: w, Clock: newSteps(), ThisMachine: page.Context{Machine: thisMac, Kind: "mac"}, Report: func(awake.View) *awake.Report { return nil }}
	in := engine.FeedInput{}
	in.Awake = nothing(in.Awake)
	p := until(t, o.Feed()(ctx, in), "the hero", func(p page.Page) bool { return p.Badge != nil })
	head := p.Sections[0].Items[0].Row
	ends := newSteps().Now().Add(22 * time.Minute).In(time.Local).Format("15:04")
	if head.Label != "Claude on linux-1 ends in 22 min" || len(head.Detail.Items) != 2 || head.Detail.Items[1].Text != "ends "+ends {
		t.Errorf("hero %q %+v, want Claude on linux-1 ending in 22 min, at %s", head.Label, head.Detail.Items, ends)
	}
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
	r := row(p, "needs-you", id(thisMac, catalog.ProviderCloudflare))
	if r.Label != "This Mac" || r.Detail.Text != "Cloudflare needs sign-in" {
		t.Errorf("the sign-in reads %q, %q, want This Mac, Cloudflare needs sign-in", r.Label, r.Detail.Text)
	}
	a := action(p, r.Verb)
	if a == nil || a.Label != "Sign in" || a.Busy != "Waiting" || a.Run == nil || !a.Run.Terminal || !slices.Equal(a.Run.Argv, []string{"accounts"}) {
		t.Fatalf("Sign in is %+v, want a terminal on station accounts, busy Waiting while it runs", a)
	}
	if a.Hud == nil || len(a.Hud.Items) != 2 || a.Hud.Items[1].Text != "station settings apply cloudflare-login on This Mac" {
		t.Errorf("Sign in's HUD is %+v, want Terminal and the login it runs, on This Mac", a.Hud)
	}
}

func TestACopyEndingSoonNeedsYouWithExtendAndTheFleetPageSharesFromItsFooter(t *testing.T) {
	p := watchAccounts(t, twoMacsSignedIn(), "")
	r := row(p, "needs-you", "short/"+id(thisMac, "linux-1", catalog.ProviderClaude))
	if r.Label != "linux-1" || r.Detail.Text != "Claude ends in 22 min" {
		t.Errorf("the copy ending reads %q, %q, want linux-1, Claude ends in 22 min", r.Label, r.Detail.Text)
	}
	if a := action(p, r.Verb); a == nil || a.Label != "Extend" || a.Run == nil || a.Run.Call != "accounts.share" || a.Run.In["With"] != "linux-1" || a.Run.In["Account"] != catalog.ProviderClaude {
		t.Errorf("Extend is %+v, want Claude shared with linux-1 again", a)
	}
	if p.Footer == nil || !slices.Equal(p.Footer.Buttons, []string{"full-list", "share-account"}) {
		t.Fatalf("the footer is %+v, want Full list in Terminal then Lend an account", p.Footer)
	}
	if full := action(p, "full-list"); full == nil || !slices.Equal(full.Run.Argv, []string{"accounts", "--all-machines"}) {
		t.Errorf("Full list in Terminal is %+v, want station accounts --all-machines", full)
	}
	if slices.Contains(rowIDs(p, "shared"), "share-account") {
		t.Error("the fleet page shares from its footer, never from a row")
	}
}
