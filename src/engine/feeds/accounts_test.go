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
	if got, want := sectionIDs(p), []string{"needs-you", "signed-in:" + thisMac, "signed-in:" + mini, "shared", "silent"}; !slices.Equal(got, want) {
		t.Fatalf("sections %v, want %v", got, want)
	}
	if got, want := rowIDs(p, "needs-you"), []string{id(thisMac, catalog.ProviderCloudflare), "short/" + id(thisMac, "linux-1", catalog.ProviderClaude)}; !slices.Equal(got, want) {
		t.Errorf("needs you %v, want Cloudflare signed out then Claude's copy under 30 min", got)
	}
	if s := p.Sections[1]; s.Title != "Signed in on This Mac" || p.Sections[2].Title != "Signed in on mac-mini" {
		t.Errorf("signed in under %q and %q, want each Mac by its name", s.Title, p.Sections[2].Title)
	}
	if got, want := rowIDs(p, "signed-in:"+thisMac), []string{id(thisMac, catalog.ProviderGitHub), id(thisMac, catalog.ProviderClaude)}; !slices.Equal(got, want) {
		t.Errorf("This Mac signed in to %v, want GitHub and Claude, git being no sign-in", got)
	}
	if r := row(p, "silent", air); r == nil || r.Cells[0].Text != "macbook-air does not answer" {
		t.Errorf("the air is %+v, want it named as not answering", r)
	}
}

func TestASignedInAccountSaysWhatItsProviderPrintedAndNothingMore(t *testing.T) {
	p := watchAccounts(t, twoMacsSignedIn(), "")
	for _, c := range []struct {
		section, id string
		want        []string
	}{
		{"signed-in:" + thisMac, id(thisMac, catalog.ProviderGitHub), []string{"jterrazz", "repo, workflow"}},
		{"signed-in:" + thisMac, id(thisMac, catalog.ProviderClaude), []string{"Max plan"}},
		{"signed-in:" + mini, id(mini, catalog.ProviderGitHub), []string{"jterrazz-bot", "repo"}},
		{"signed-in:" + mini, id(mini, catalog.ProviderClaude), []string{"Max plan"}},
	} {
		r := row(p, c.section, c.id)
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
	if got, want := sectionIDs(p), []string{"signed-in:" + mini}; !slices.Equal(got, want) {
		t.Errorf("the mini's accounts are %v, want %v", got, want)
	}
}

func TestAWorkspacesAccountsAreTheCopiesItsMacSharedWithIt(t *testing.T) {
	p := watchAccounts(t, twoMacsSignedIn(), "linux-1")
	if got, want := sectionIDs(p), []string{"needs-you", "shared"}; !slices.Equal(got, want) {
		t.Fatalf("linux-1's accounts are %v, want %v", got, want)
	}
	if s := p.Sections[1]; s.Title != "Shared with linux-1" || len(s.Items) != 2 {
		t.Errorf("linux-1's copies are %q with %d rows, want its two", s.Title, len(s.Items))
	}
	w := twoMacsSignedIn()
	w.beside = []machines.VM{{Name: "review-bot", Mac: mini, Lifetime: machines.Throwaway, State: machines.Running}}
	bot := watchAccounts(t, w, "review-bot")
	if s := bot.Sections; len(s) != 1 || s[0].Title != "Shared with review-bot" || s[0].Items[0].Empty == nil || s[0].Items[0].Empty.Text != "No account shared with review-bot" {
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
