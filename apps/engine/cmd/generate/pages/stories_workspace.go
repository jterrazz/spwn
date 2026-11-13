package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	pagemenubar "github.com/jterrazz/studio-station/apps/engine/pages/menubar"
)

func workspaceWorlds() []World {
	return []World{
		{Name: "machines.agents.two", Edit: docsBotOnLinux1, Pages: []string{"machine.linux-1"}},
		{Name: "machines.agents.started", Edit: apiStarted, Pages: []string{"machine.api"}},
		{Name: "machines.agents.replied", Edit: reviewBotReplied, Pages: []string{"agent.review-bot"}},
		{Name: "machines.agents.merged", Edit: reviewBotMerged, Pages: []string{"agent.review-bot"}},
		{Name: "machines.agents.unhooked", Edit: linux1Unhooked, Pages: []string{"machine.linux-1"}},
	}
}

func ago(d time.Duration) time.Time { return storyNow.Add(-d) }

// value: STORIES' shared fixture since A2: each agent an object of its
// own on the VM of its name, one Herdr terminal each.
func sceneAgents() []agents.Agent {
	return []agents.Agent{
		{Name: "linux-1", VM: "linux-1", Mac: proName, App: "claudecode", Session: "s-linux-1", Terminal: "t1", State: agents.Working, Task: "builds the landing page", Since: ago(40 * time.Minute),
			Steps: []agents.Step{
				{At: ago(7 * time.Minute), Verb: catalog.AgentEdit, Object: "the pricing section"},
				{At: ago(3 * time.Minute), Verb: catalog.AgentEdit, Object: "src/pages/index.astro", Diff: &agents.Diff{Plus: 86, Minus: 12}},
				{At: ago(10 * time.Second), Verb: catalog.AgentTest, Tests: &agents.Tests{Pass: 41, Total: 58}},
			}},
		{Name: "agent-7f", VM: "agent-7f", Mac: proName, App: "claudecode", Session: "s-agent-7f", State: agents.Finished, Task: "write the changelog", Ended: ago(12 * time.Minute), Reason: "tests pass",
			Steps: []agents.Step{
				{At: ago(31 * time.Minute), Verb: catalog.AgentEdit, Object: "CHANGELOG.md", Diff: &agents.Diff{Plus: 42, Minus: 3}},
				{At: ago(18 * time.Minute), Verb: catalog.AgentTest, Tests: &agents.Tests{Pass: 58, Total: 58}},
				{At: ago(13 * time.Minute), Verb: catalog.AgentCommit},
			}},
		{Name: "linux-3", VM: "linux-3", Mac: miniMac, App: "claudecode", Session: "s-linux-3", Terminal: "t1", State: agents.Working, Task: "refresh the auth token",
			Steps: []agents.Step{
				{At: ago(5 * time.Minute), Verb: catalog.AgentTest, Tests: &agents.Tests{Pass: 55, Total: 58}},
				{At: ago(20 * time.Second), Verb: catalog.AgentEdit, Object: "src/auth/refresh.ts"},
			}},
		{Name: "review-bot", VM: "review-bot", Mac: miniMac, App: "claudecode", Session: "s-review-bot", Terminal: "t1", State: agents.Waiting, Task: "fixes the token refresh", Since: ago(2 * time.Hour),
			Ask: &agents.Ask{Title: "PR #42 is ready, tests pass", Detail: "I moved the refresh into its own commit. The retry path changes how a 401 is handled: check it before I merge.", At: ago(6 * time.Minute)},
			PR:  &agents.PR{Number: 42, URL: "https://github.com/jterrazz/app/pull/42"}, Change: &agents.Change{Files: 6, Plus: 214, Minus: 88},
			Steps: []agents.Step{{At: ago(7 * time.Minute), Verb: catalog.AgentOpenPR}}},
	}
}

func agentIn(f *Scene, name string) *agents.Agent {
	for i := range f.Agents {
		if f.Agents[i].Name == name {
			return &f.Agents[i]
		}
	}
	panic("no agent " + name + " in the scene")
}

func docsBotOnLinux1(f *Scene) {
	f.Agents = append(f.Agents, agents.Agent{Name: "docs-bot", VM: "linux-1", Mac: proName, App: "claudecode", Session: "s-docs-bot", Terminal: "t2", State: agents.Working, Task: "writes the changelog", Since: ago(12 * time.Minute)})
}

func apiStarted(f *Scene) {
	for i, vm := range f.All.Macs[0].VMs {
		if vm.Name == "api" {
			f.All.Macs[0].VMs[i].State, f.All.Macs[0].VMs[i].BootAt, f.All.Macs[0].VMs[i].Guest = model.Running, ago(10*time.Second), "connected"
		}
	}
	f.Samples["api"] = readingsAt(storyNow, 8*gb, 40e9, 36e9)
}

func reviewBotReplied(f *Scene) {
	a := agentIn(f, "review-bot")
	a.State, a.Ask, a.Since = agents.Working, nil, ago(10*time.Second)
	a.Steps = append(a.Steps, agents.Step{At: ago(5 * time.Second), Verb: catalog.AgentRead, Object: "your reply"})
}

func reviewBotMerged(f *Scene) {
	a := agentIn(f, "review-bot")
	a.State, a.Ask, a.Ended, a.Reason = agents.Finished, nil, ago(5*time.Second), "merged PR #42"
}

func linux1Unhooked(f *Scene) {
	a := agentIn(f, "linux-1")
	a.State, a.Steps, a.Since = agents.Running, nil, time.Time{}
}

func (f Scene) agent(name string) pagemenubar.Agent {
	return pagemenubar.Agent{Name: name, All: &f.All, Agents: f.Agents, Now: storyNow, Asks: Levels}
}
