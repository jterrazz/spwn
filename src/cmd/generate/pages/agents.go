package pages

import (
	"time"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/page"
	pageagents "github.com/jterrazz/studio-station/src/pages/agents"
)

func workspaceContext() page.Context { return page.Context{Kind: "workspace", Machine: "review-bot"} }

// value: the agents of a workspace, one in each state its hook says: one
// waiting on its PR, one at work on a test run, one finished.
func agentsKept() []agents.Agent {
	at := time.Date(2026, 11, 7, 9, 24, 0, 0, time.UTC)
	return []agents.Agent{
		{Name: "review-bot", VM: "review-bot", App: "claudecode", Session: "review-bot", State: agents.Waiting, Task: "Review the auth refresh", Folder: "/home/admin/app",
			Ask: &agents.Ask{Title: "PR #42 is ready", Detail: "I moved the refresh into its own commit.", At: at},
			PR:  &agents.PR{Number: 42, URL: "https://github.com/jterrazz/app/pull/42"}, Change: &agents.Change{Files: 6, Plus: 214, Minus: 88}},
		{Name: "review-bot" + agents.Nth + "2", VM: "review-bot", App: "claudecode", Session: "s-2", State: agents.Working, Task: "Build the landing page", Folder: "/home/admin/site",
			Steps: []agents.Step{{At: at, Verb: catalog.AgentEdit, Object: "src/pages/index.astro", Diff: &agents.Diff{Plus: 86, Minus: 12}}, {At: at, Verb: catalog.AgentTest, Tests: &agents.Tests{Pass: 41, Total: 58}}}},
		{Name: "review-bot" + agents.Nth + "3", VM: "review-bot", App: "codex", Session: "t-1", State: agents.Finished, Task: "Write the changelog", Reason: "merged PR #40", Ended: at},
	}
}

func agentsPages() []Fixture {
	kept := agentsKept()
	return states("agents",
		pending(pageagents.List(kept, workspaceContext())),
		pageagents.List(nil, workspaceContext()),
		failure("agents", "agents", "the agents' journal could not be read", "station agents"),
		pageagents.List(kept, workspaceContext()))
}
