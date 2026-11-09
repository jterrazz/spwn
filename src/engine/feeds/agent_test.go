package feeds_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/engine/feeds"
	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

func agentWorld() *world {
	linux := machines.VM{Name: "linux-1", Mac: thisMac, Lifetime: machines.Throwaway, State: machines.Running, Guest: "connected"}
	at := time.Date(2026, 11, 7, 10, 24, 0, 0, time.UTC)
	return &world{infra: []machines.VM{infraAt(machines.Running, 0)}, here: []machines.VM{linux}, asked: map[string][]string{}, activity: map[string]uint64{},
		agents: []agents.Agent{
			{Name: "linux-1", VM: "linux-1", Mac: thisMac, App: "claudecode", State: agents.Waiting, Ask: &agents.Ask{Title: "PR #42 is ready", At: at}, PR: &agents.PR{Number: 42, URL: "https://github.com/jterrazz/app/pull/42"}},
			{Name: "docs-bot", VM: "linux-1", Mac: thisMac, App: "claudecode", State: agents.Working, Task: "writes the changelog"},
		}}
}

func TestAWorkspacesPageListsEachAgentItHostsByItsName(t *testing.T) {
	pages := watchWith(t, agentWorld(), newSteps(), engine.FeedInput{Subject: "linux-1"})
	p := until(t, pages, "linux-1's two agents", func(p page.Page) bool {
		return row(p, "agents", "linux-1") != nil && row(p, "agents", "docs-bot") != nil
	})
	if nav := row(p, "agents", "docs-bot").Nav; nav != "menubar.agent:docs-bot" {
		t.Fatalf("docs-bot's row opens %q", nav)
	}
}

func TestAnAgentsPageTurnsToItsStepsOnceItsReplyIsSent(t *testing.T) {
	w := agentWorld()
	refresh := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	pages := feeds.Menubar{Calls: w, Clock: newSteps()}.Agent()(ctx, engine.FeedInput{Subject: "linux-1", Refresh: refresh})
	until(t, pages, "linux-1 waiting on you", func(p page.Page) bool { return slices.Contains(sectionIDs(p), "wait") })
	w.mu.Lock()
	w.agents[0].State, w.agents[0].Ask = agents.Working, nil
	w.mu.Unlock()
	refresh <- struct{}{}
	p := until(t, pages, "linux-1 at work again", func(p page.Page) bool { return slices.Contains(sectionIDs(p), "now") })
	if slices.Contains(sectionIDs(p), "wait") {
		t.Fatalf("linux-1 still waits: %q", sectionIDs(p))
	}
}

func TestAnAgentNoMachineHoldsIsSaidSo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	pages := feeds.Menubar{Calls: agentWorld(), Clock: newSteps()}.Agent()(ctx, engine.FeedInput{Subject: "ghost"})
	until(t, pages, "no agent named ghost", func(p page.Page) bool { return p.Error != nil && p.Error.Message == "no agent is named ghost" })
}

// invariant: SPEC A5 Clean up (A2): Home forgets a finished agent by its
// own name, read from the fleet's agents, though it shares its VM with a
// working one.
func TestHomesCleanUpForgetsAFinishedAgentByItsOwnName(t *testing.T) {
	w := agentWorld()
	w.agents[0].State, w.agents[0].Ask = agents.Working, nil
	w.agents[1].State = agents.Finished
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	o := feeds.Menubar{Calls: w, Clock: newSteps(), ThisMachine: page.Context{Machine: thisMac}, Report: func(awake.View) *awake.Report { return nil }}
	pages := o.Feed()(ctx, engine.FeedInput{Changed: make(chan struct{}), Refresh: make(chan struct{}), Awake: func() awake.View { return awake.View{} }})
	until(t, pages, "Clean up forgetting docs-bot", func(p page.Page) bool {
		for _, a := range p.Actions {
			if a.ID != "cleanup" || a.Run == nil {
				continue
			}
			for _, r := range append([]page.Run{*a.Run}, a.Run.And...) {
				if r.Call == "agents.forget" && r.In["Agent"] == "docs-bot" && r.Target != nil && r.Target.Name == "linux-1" {
					return true
				}
			}
		}
		return false
	})
}
