package pages

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

func workspaceWorlds() []World {
	return []World{
		{Name: "machines.started", Edit: apiStarted, Pages: []string{"machine.api"}},
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

func apiStarted(f *Scene) {
	for i, vm := range f.All.Macs[0].VMs {
		if vm.Name == "api" {
			f.All.Macs[0].VMs[i].State, f.All.Macs[0].VMs[i].BootAt, f.All.Macs[0].VMs[i].Guest = model.Running, ago(10*time.Second), "connected"
		}
	}
	f.Samples["api"] = readingsAt(storyNow, 8*gb, 40e9, 36e9)
}
