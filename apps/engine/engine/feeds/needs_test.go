package feeds_test

import (
	"context"
	"slices"
	"testing"

	accountscalls "github.com/jterrazz/studio-station/apps/engine/accounts/calls"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/feeds"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	overviewcalls "github.com/jterrazz/studio-station/apps/engine/overview/calls"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemenubar "github.com/jterrazz/studio-station/apps/engine/pages/menubar"
)

func watchNeeds(t *testing.T, w *world, subject string) page.Page {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	pages := feeds.Menubar{Calls: w, Clock: newSteps(), ThisMachine: page.Context{Machine: thisMac, Kind: "computer"}}.Needs()(ctx, engine.FeedInput{Subject: subject})
	return until(t, pages, "the needs read", func(p page.Page) bool { return p.State != page.Loading })
}

// invariant: SPEC B5.7: the needs page is read from the fleet, every Mac's
// accounts and Overview, each through its own engine: infra down on
// mac-mini comes first, then linux-1 waiting on This Mac, then the copy of
// Claude ending; This Mac's updates are upkeep; a Mac that does not answer
// is asked nothing.
func TestTheNeedsPageReadsEveryMacThroughItsEngineAndRanksWhatNeedsYou(t *testing.T) {
	w := twoMacsSignedIn()
	w.infra = []machines.VM{infraAt(machines.Stopped, 5)}
	w.infra[0].Restarts.Looping = true
	w.here[0].Agent = &machines.AgentView{State: "waiting", Ask: "PR #42 is ready"}
	w.updates = true

	p := watchNeeds(t, w, "")

	if p.Error != nil {
		t.Fatalf("the needs page failed: %s", p.Error.Message)
	}
	if got, want := rowIDs(p, pagemenubar.SectionNeeds), []string{"down:infra", "waiting:linux-1", "ending:" + thisMac + "/linux-1/" + catalog.ProviderClaude}; !slices.Equal(got, want) {
		t.Errorf("needs %v, want %v", got, want)
	}
	if got := rowIDs(p, pagemenubar.SectionUpkeep); len(got) != 1 {
		t.Errorf("upkeep %v, want This Mac's updates alone", got)
	}
	for _, read := range []string{accountscalls.List.Name, overviewcalls.Read.Name} {
		if got := w.askedOf(read); !slices.Equal(got, []string{thisMac, mini}) {
			t.Errorf("%s asked %v, want This Mac then mac-mini, never the silent air", read, got)
		}
	}

	scoped := watchNeeds(t, w, "mac-mini")
	if got := rowIDs(scoped, pagemenubar.SectionNeeds); !slices.Equal(got, []string{"down:infra"}) {
		t.Errorf("mac-mini's needs %v, want infra alone", got)
	}
}
