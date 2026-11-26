package pages

import (
	"github.com/jterrazz/studio-station/apps/engine/agents"
	pageagents "github.com/jterrazz/studio-station/apps/engine/pages/agents"
)

// value: the agents of a project on This Mac: one on the default harness,
// one of a team on the other.
func agentList() []agents.Agent {
	return []agents.Agent{
		{Name: "neo", Description: "Reviews the pull requests of the web team.", Team: "web", Harness: "claude-code"},
		{Name: "trinity", Description: "Keeps the release notes.", Harness: "codex"},
	}
}

func agentPages() []Fixture {
	return states("agents",
		pending(pageagents.List(agentList(), thisMacContext())),
		pageagents.List(nil, thisMacContext()),
		failure("agents", "agents", "the agents of ~/Developer/site cannot be read", "station agents"),
		pageagents.List(agentList(), thisMacContext()),
	)
}
